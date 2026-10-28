package main

import (
	"time"

	"github.com/bcicen/ctop/config"
	"github.com/bcicen/ctop/cwidgets/single"
	"github.com/bcicen/ctop/terminal"
	"github.com/bcicen/ctop/widgets"
	ui "github.com/gizak/termui"
)

func ShowConnError(err error) (exit bool) {
	ui.Clear()
	terminal.ResetHandlers()
	defer terminal.ResetHandlers()

	setErr := func(err error) {
		errView.Append(err.Error())
		errView.Append("attempting to reconnect...")
		ui.Render(errView)
	}

	HandleKeys("exit", func() {
		exit = true
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(ui.Event) {
		_, err := cursor.RefreshContainers()
		if err == nil {
			ui.StopLoop()
			return
		}
		setErr(err)
	})

	terminal.HandleResize(func(e ui.Event) {
		errView.Resize()
		ui.Clear()
		ui.Render(errView)
		log.Infof("RESIZE")
	})

	errView.Resize()
	setErr(err)
	ui.Loop()
	return exit
}

func RedrawRows(clr bool, overlays ...ui.Bufferer) {
	width, height := terminal.Size()
	// reinit body rows
	cGrid.Clear()

	// build layout
	y := 1
	if config.GetSwitchVal("enableHeader") {
		header.SetCount(cursor.Len())
		header.SetFilter(config.GetVal("filterStr"))
		y += header.Height()
	}

	var buildPanel *widgets.BuildPanel
	if buildMonitor != nil {
		rows, warning := buildMonitor.Snapshot()
		// Share the screen with containers, reserving space for their header and rows.
		buildPanel = widgets.NewBuildPanel(rows, warning, width-2, (height-y-4)/2, time.Now())
		buildPanel.Y = y
		y += buildPanel.Height
	}
	// Clear when the build section changes height, including when the last build ends.
	clr = clr || cGrid.Y != y
	cGrid.SetY(y)

	for _, c := range cursor.filtered {
		cGrid.AddRows(c.Widgets)
	}

	cursor.ScrollPage()
	cGrid.Align()
	var parts []ui.Bufferer
	if config.GetSwitchVal("enableHeader") {
		parts = append(parts, header)
	}
	if buildPanel != nil && buildPanel.Height > 0 {
		parts = append(parts, buildPanel)
	}
	parts = append(parts, cGrid)
	parts = append(parts, overlays...)
	ui.Render(newRenderFrame(clr, parts...))
}

func SingleView() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	ui.Clear()
	terminal.ResetHandlers()
	defer terminal.ResetHandlers()

	ex := single.NewSingle()
	c.SetUpdater(ex)

	ex.Align()
	ui.Render(ex)

	HandleKeys("up", ex.Up)
	HandleKeys("down", ex.Down)
	ui.Handle("/sys/kbd/", func(ui.Event) { ui.StopLoop() })

	ui.Handle("/timer/1s", func(ui.Event) { ui.Render(ex) })
	terminal.HandleResize(func(e ui.Event) {
		width, _ := terminal.Size()
		ex.SetWidth(width)
		ex.Align()
		log.Infof("resize: width=%v max-rows=%v", ex.Width, cGrid.MaxRows())
	})

	ui.Loop()
	c.SetUpdater(c.Widgets)
	return nil
}

func RefreshDisplay(overlays ...ui.Bufferer) error {
	// skip display refresh during scroll
	if !cursor.isScrolling {
		needsClear, err := cursor.RefreshContainers()
		if err != nil {
			return err
		}
		RedrawRows(needsClear, overlays...)
	}
	return nil
}

func Display() bool {
	var menu MenuFn
	var connErr error

	width, _ := terminal.Size()
	cGrid.SetWidth(width)
	ui.DefaultEvtStream.Hook(logEvent)

	// initial draw
	header.Align()
	status.Align()
	cursor.RefreshContainers()
	RedrawRows(true)

	HandleKeys("up", cursor.Up)
	HandleKeys("down", cursor.Down)

	HandleKeys("pgup", cursor.PgUp)
	HandleKeys("pgdown", cursor.PgDown)

	HandleKeys("exit", ui.StopLoop)
	HandleKeys("help", func() {
		menu = HelpMenu
		ui.StopLoop()
	})

	ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
		menu = ContainerMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/<left>", func(ui.Event) {
		menu = LogMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/<right>", func(ui.Event) {
		menu = SingleView
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/l", func(ui.Event) {
		menu = LogMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/e", func(ui.Event) {
		menu = ExecShell
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/w", func(ui.Event) {
		menu = OpenInBrowser()
	})
	ui.Handle("/sys/kbd/o", func(ui.Event) {
		menu = SingleView
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/a", func(ui.Event) {
		config.Toggle("allContainers")
		connErr = RefreshDisplay()
		if connErr != nil {
			ui.StopLoop()
		}
	})
	ui.Handle("/sys/kbd/D", func(ui.Event) {
		dumpContainer(cursor.Selected())
	})
	ui.Handle("/sys/kbd/f", func(ui.Event) {
		menu = FilterMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/H", func(ui.Event) {
		config.Toggle("enableHeader")
		RedrawRows(true)
	})
	ui.Handle("/sys/kbd/r", func(e ui.Event) {
		config.Toggle("sortReversed")
	})
	ui.Handle("/sys/kbd/s", func(ui.Event) {
		menu = SortMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/c", func(ui.Event) {
		menu = ColumnsMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/S", func(ui.Event) {
		path, err := config.Write()
		if err == nil {
			log.Statusf("wrote config to %s", path)
		} else {
			log.StatusErr(err)
		}
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		if log.StatusQueued() {
			ui.StopLoop()
		}
		connErr = RefreshDisplay()
		if connErr != nil {
			ui.StopLoop()
		}
	})

	terminal.HandleResize(func(e ui.Event) {
		width, _ := terminal.Size()
		header.Align()
		status.Align()
		cursor.ScrollPage()
		cGrid.SetWidth(width)
		log.Infof("resize: width=%v max-rows=%v", cGrid.Width, cGrid.MaxRows())
		RedrawRows(true)
	})

	ui.Loop()

	if connErr != nil {
		return ShowConnError(connErr)
	}

	if log.StatusQueued() {
		for sm := range log.FlushStatus() {
			if sm.IsError {
				status.ShowErr(sm.Text)
			} else {
				status.Show(sm.Text)
			}
		}
		return false
	}

	if menu != nil {
		for menu != nil {
			menu = menu()
		}
		return false
	}

	return true
}
