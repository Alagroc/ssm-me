package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTextModal returns a scrollable TextView that dismisses on Esc or q.
func newTextModal(app *App, text, pageName string) *tview.TextView {
	tv := tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	tv.SetText(text)
	tv.SetBorder(true)

	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			app.pages.RemovePage(pageName)
			return nil
		}
		return event
	})
	return tv
}
