package streams

import (
	"github.com/gdamore/tcell"
)

// Tcell is an output stream used by the DockerCli to write stats output using
// tcell go package.
type Tcell struct {
	screen tcell.Screen
	width  int
	x      int
	y      int
}

// Screen returns the underlying tcell.Screen object of type Tcell.
func (tc *Tcell) Screen() tcell.Screen {
	return tc.screen
}

// Init initializes the underlying screen and set width member to screen width.
// If initialization failed an error is returned.
func (tc *Tcell) Init() error {
	if err := tc.screen.Init(); err != nil {
		return err
	}

	tc.width, _ = tc.screen.Size()

	return nil
}

// Resize set the width member to the given argument.
// It needs to be done when the windows where docker stats runs is resized.
func (tc *Tcell) Resize(width int) {
	tc.width = width

	/*
	 * When we need to resize it is better to totally empty the buffer so no
	 * ancient characters are still displayed.
	 */
	tc.screen.Clear()
	tc.screen.Sync()
}

// Display synchronizes the underlying screen object so its buffer is printed.
func (tc *Tcell) Display() {
	tc.x = 0
	tc.y = 0

	tc.screen.Sync()
}

/*
 * Write adds byte array to buffer of underlying screen object.
 * If the buffer is longer than width it will be sliced to be printed on
 * multiple lines.
 * It returns the number of bytes written to the buffer to its length and no
 * errors are returned.
 */
func (tc *Tcell) Write(p []byte) (int, error) {
	screen := tc.screen
	length := len(p)

	for i := 0; i < length; i++ {

		screen.SetContent(tc.x, tc.y, rune(p[i]), nil, tcell.StyleDefault)

		// Pass to next x since we pass to next character.
		tc.x++

		/*
		 * If p[i] is '\n' we need to pass to next line.
		 * But we also need to do it if x is divisible by width because it means
		 * that p is longer than screen width.
		 */
		if p[i] == '\n' || tc.x > 0 && tc.x%tc.width == 0 {
			// So we increase y to effectively pass to next line.
			tc.y++

			// And we reset x to begin the new line from 0.
			tc.x = 0
		}
	}

	return length, nil
}

// NewTcell returns a new Tcell object.
func NewTcell() *Tcell {
	screen, err := tcell.NewScreen()

	if err != nil {
		return nil
	}

	return &Tcell{screen: screen, x: 0, y: 0}
}
