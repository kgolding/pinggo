package main

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Result struct {
	LastSeen time.Time
	Changed  time.Time
	Active   bool
	IP       string
	Name     string
	Widget   *IPWidget
}

type App struct {
	// GUI vars
	app fyne.App
	win fyne.Window

	// App vars
	ip       string
	autoscan bool
	results  map[string]*Result

	// Widgets
	cResults  *fyne.Container
	wStatus   *widget.Label
	wScan     *widget.Button
	wProgress *widget.ProgressBar

	// Other
	cancelScan context.CancelFunc
	sync.Mutex
}

func main() {
	a := &App{
		app:      app.New(),
		results:  make(map[string]*Result),
		autoscan: true,
	}
	a.win = a.app.NewWindow("PingGo")
	a.win.SetIcon(resIconPng)
	a.win.SetContent(a.mainWindow())
	a.win.Resize(fyne.NewSize(550, 500))
	a.win.CenterOnScreen()

	a.win.ShowAndRun()
}

// Clear the main results
func (a *App) Clear() {
	a.Lock()
	a.results = make(map[string]*Result)
	a.Unlock()
	a.cResults.Objects = nil
}

// Scan starts a scan, cancelling a current scan if applicable
func (a *App) Scan() {
	if a.ip == "" {
		return
	}

	go func() {
		ipList, err := Hosts(a.ip)
		if err != nil {
			a.wStatus.SetText("Error: " + err.Error())
			return
		}

		if a.cancelScan != nil {
			a.cancelScan()
		}
		var ctx context.Context

		ctx, a.cancelScan = context.WithCancel(context.Background())
		a.wStatus.SetText("Scanning: " + a.ip)
		results, err := a.startScan(ctx, ipList)
		a.wStatus.SetText("")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			a.wStatus.SetText("Error: " + err.Error())
			return
		}

		now := time.Now()
		a.Lock()
		// Add/update results
		for i, item := range results {
			// nowi is the time the scan starts incremented by a ms to ensure the results are sorted neatly
			nowi := now.Add(time.Millisecond * time.Duration(i))
			if r, ok := a.results[item.IP.String()]; ok { // Already in results
				if !r.Active {
					r.Changed = nowi
				}
				r.LastSeen = nowi
				r.Active = true
				r.Widget.active = true
			} else { // Add to cache
				a.results[item.IP.String()] = &Result{
					Active:   true,
					LastSeen: nowi,
					Changed:  nowi,
					IP:       item.IP.String(),
					// Name:     item.Name,
					Widget: newIPWidget(item.IP.String()),
				}
			}
		}
		// Mark results that are no longer pinging
		for ip, item := range a.results {
			exists := false
			for _, r := range results {
				if r.IP.String() == ip {
					exists = true
				}
			}
			if !exists {
				if item.Active {
					item.Active = false
					item.Changed = now.Add(-time.Second) // sorts inactive below active
					item.Widget.active = false
				}
			}
		}

		// Create a copy of the results map as an array to sort
		arr := make([]*Result, 0)
		for _, r := range a.results {
			arr = append(arr, r)
		}
		sort.Slice(arr, func(i, j int) bool {
			return arr[i].Changed.After(arr[j].Changed)
		})

		// Copy the sorted results array to the fyne grid widget
		a.cResults.Objects = make([]fyne.CanvasObject, len(arr))
		for i, r := range arr {
			a.cResults.Objects[i] = r.Widget
		}
		autoscan := a.autoscan
		a.Unlock()

		a.cResults.Refresh()
		if autoscan {
			// Have a break before starting the next scan
			time.Sleep(time.Millisecond * 100)
			a.Scan()
		}
	}()
}

// mainWindows creates the main window
func (a *App) mainWindow() fyne.CanvasObject {
	nets := GetIPv4NonLocalInterfaces()
	wSelectNet := widget.NewSelect(nets, func(s string) {
		a.ip = s
		a.Clear()
		a.Scan()
	})
	// Whilst this is never seen, it ensures the widget is wider
	wSelectNet.PlaceHolder = "-- Select network -- "

	a.wScan = widget.NewButton("Scan", func() {
		a.Scan()
	})

	wAutoscan := widget.NewCheck("Autoscan", func(v bool) {
		a.Lock()
		a.autoscan = v
		a.Unlock()
		if v {
			a.Scan()
			a.wScan.Disable()
		} else {
			a.wScan.Enable()
		}
	})
	wAutoscan.SetChecked(true)

	top := fyne.NewContainerWithLayout(
		layout.NewHBoxLayout(),
		wSelectNet,
		layout.NewSpacer(),
		a.wScan,
		wAutoscan,
	)

	a.wStatus = widget.NewLabel("")
	a.wProgress = widget.NewProgressBar()

	wAuthor := widget.NewLabel("github.com/kgolding/pinggo")

	bottom := fyne.NewContainerWithLayout(
		layout.NewHBoxLayout(),
		a.wStatus,
		layout.NewSpacer(),
		// wOpenFiles,
		wAuthor,
		a.wProgress,
	)

	// Set the grid cells to be wide enough for a long IP address
	ts := fyne.MeasureText("X666.666.666.666X", theme.TextSize(), a.wStatus.TextStyle)
	ts.Height += theme.Padding() * 3

	a.cResults = fyne.NewContainerWithLayout(
		NewGridWrapJustifiedLayout(ts),
	)

	// Auto select the first network, which in turn triggers scanning
	if len(nets) > 0 {
		wSelectNet.SetSelectedIndex(0)
	}

	return fyne.NewContainerWithLayout(
		layout.NewBorderLayout(
			top, bottom, nil, nil,
		),
		top,
		bottom,
		container.NewScroll(a.cResults),
	)
}

func (a *App) startScan(ctx context.Context, ipList []net.IP) ([]*IPInfo, error) {
	progress := float64(0)
	var progressMutex sync.Mutex
	a.wProgress.Max = float64(len(ipList))
	a.wProgress.SetValue(progress)

	ctx, cancel := context.WithCancel(ctx)

	// This can be increased to speed up the scan
	maxOpenFiles := 64

	jobCh := make(chan struct{}, maxOpenFiles)
	var wg sync.WaitGroup
	var results []*IPInfo

	for _, ip := range ipList {
		// Check the contect hasn't been cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		wg.Add(1)
		jobCh <- struct{}{}
		go func(ip net.IP) {
			defer func() {
				progressMutex.Lock()
				progress++
				a.wProgress.SetValue(progress)
				progressMutex.Unlock()
				wg.Done()
			}()
			args := []string{"-w", "1", "-c", "1", ip.String()}
			if runtime.GOOS == "windows" {
				args = []string{"-w", "1000", "-n", "1", ip.String()}
			}
			cmd := exec.CommandContext(ctx, "ping", args...)
			_, err := cmd.Output()
			if err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					// The program has exited with an exit code != 0 which happens
					// when the remote IP doesn't exist
				} else {
					// Dodgy err... cancel as we might have hit the max open files
					cancel()
					a.wStatus.SetText("Error: " + err.Error())
				}
			} else {
				r := &IPInfo{
					IP: ip,
				}
				// for port, _ := range TCPPorts {
				// 	wg.Add(1)
				// 	go func(port int) {
				// 		defer wg.Done()
				// 		if c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip.String(), port), time.Second); err == nil {
				// 			r.AddTCPPort(port)
				// 			c.Close()
				// 		}
				// 	}(port)
				// }
				results = append(results, r)
			}
			<-jobCh
		}(ip)
		time.Sleep(time.Millisecond * 5)
	}

	wg.Wait()

	sort.Slice(results, func(a, b int) bool {
		return binary.BigEndian.Uint32(results[a].IP) < binary.BigEndian.Uint32(results[b].IP)
	})

	return results, ctx.Err()

}

func Hosts(cidr string) ([]net.IP, error) {
	// convert string to IPNet struct
	_, ipv4Net, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	// convert IPNet struct mask and address to uint32
	mask := binary.BigEndian.Uint32(ipv4Net.Mask)
	start := binary.BigEndian.Uint32(ipv4Net.IP)

	// find the final address
	finish := (start & mask) | (mask ^ 0xffffffff)

	var ips []net.IP

	// loop through addresses as uint32
	for i := start + 1; i <= finish-1; i++ {
		// convert back to net.IP
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		ips = append(ips, ip)
	}
	return ips, nil
}
