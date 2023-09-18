package main

import (
	"flag"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	posRight = iota - 2
	posDown
	_
	posUp
	posLeft
)

type gameConf struct {
	level int
	debug bool
}
type appConfig struct {
	screen        tcell.Screen
	defStyle      tcell.Style
	speed         time.Duration
	conf          gameConf
	curDirection  int
	nextDirection int
	curSnackBody  []position
	applePos      position
	gameExit      chan bool
	wg            sync.WaitGroup
	random        *rand.Rand
	score         int
	xSize         int
	ySize         int
}
type position struct {
	x int
	y int
}

func main() {
	app := gameInit()
	defer app.screen.Fini()
	app.wg.Add(2)
	go app.keyPressEventListener()
	go app.snackMove()
	app.wg.Wait()
}
func gameInit() *appConfig {
	app := &appConfig{}

	flag.IntVar(&app.conf.level, "level", 5, "game difficulty 1 to 10")
	flag.BoolVar(&app.conf.debug, "debug", false, "debug mode (through wall)")
	flag.Parse()

	var err error
	app.screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = app.screen.Init()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	app.defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	app.screen.SetStyle(app.defStyle)
	app.xSize, app.ySize = app.screen.Size()

	app.speed = time.Millisecond * time.Duration(405-app.conf.level*40)
	app.random = rand.New(rand.NewSource(time.Now().UnixNano()))

	app.gameExit = make(chan bool)

	app.curDirection = posUp
	app.nextDirection = posUp

	app.curSnackBody = make([]position, 5)
	for i := range app.curSnackBody {
		app.curSnackBody[i].x = app.xSize / 2
		app.curSnackBody[i].y = app.ySize/2 - 2 + i
	}

	app.genNewApple()

	return app
}
func (app *appConfig) genNewApple() {
	app.applePos = position{
		x: app.random.Intn(app.xSize-2) + 1,
		y: app.random.Intn(app.ySize-2) + 1,
	}
}
func (app *appConfig) keyPressEventListener() {
	defer app.wg.Done()
	for {
		select {
		case <-app.gameExit:
			return
		default:
			switch ev := app.screen.PollEvent().(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyUp:
					app.nextDirection = posUp
				case tcell.KeyDown:
					app.nextDirection = posDown
				case tcell.KeyLeft:
					app.nextDirection = posLeft
				case tcell.KeyRight:
					app.nextDirection = posRight
				case tcell.KeyEsc:
					fallthrough
				case tcell.KeyCtrlC:
					app.quitGame()
				}
			}
		}
	}
}

func (app *appConfig) snackMove() {
	defer app.wg.Done()
	for {
		select {
		case <-app.gameExit:
			return
		default:
			eatTheApple := app.eatTheAppleChecker()

			// if current direction is up can't turn into down
			if abs(app.curDirection) != abs(app.nextDirection) {
				app.curDirection = app.nextDirection
			}
			getCurrentPos := app.curDirection
			newSnack := make([]position, len(app.curSnackBody))

			newSnack[0] = app.curSnackBody[0]
			switch getCurrentPos {
			case posUp:
				newSnack[0].y--
			case posDown:
				newSnack[0].y++
			case posLeft:
				newSnack[0].x--
			case posRight:
				newSnack[0].x++
			}

			// check hit the body
			if !app.conf.debug {
				for _, pos := range app.curSnackBody {
					if newSnack[0] == pos {
						app.quitGame()
					}
				}
			}
			newHead, hit := app.hitTheWallChecker(newSnack[0])

			if !app.conf.debug && hit {
				app.quitGame()
			} else {
				newSnack[0] = newHead
			}

			for i := 1; i < len(newSnack); i++ {
				newSnack[i] = app.curSnackBody[i-1]
			}
			if eatTheApple {
				app.screen.Beep()
				newSnack = append(newSnack, app.curSnackBody[len(app.curSnackBody)-1])
				app.genNewApple()
				app.score += 10
				app.scoreChecker()
			}
			app.curSnackBody = newSnack

			app.render(app.curSnackBody)
			time.Sleep(app.speed)
		}

	}
}
func (app *appConfig) eatTheAppleChecker() bool {
	return app.curSnackBody[0].x == app.applePos.x && app.curSnackBody[0].y == app.applePos.y
}
func (app *appConfig) hitTheWallChecker(snackHead position) (position, bool) {
	newPosition := snackHead
	switch {
	case snackHead.x == 0:
		newPosition.x = app.xSize - 2
	case snackHead.x == app.xSize-1:
		newPosition.x = 1
	case snackHead.y == 0:
		newPosition.y = app.ySize - 2
	case snackHead.y == app.ySize-1:
		newPosition.y = 1
	}
	if newPosition != snackHead {
		return newPosition, true
	}
	return snackHead, false
}
func (app *appConfig) scoreChecker() {
	if app.score%100 == 0 && app.conf.level != 10 {
		app.conf.level++
		app.speed = time.Millisecond * time.Duration(405-app.conf.level*40)
	}
}
func (app *appConfig) render(snackPosition []position) {
	// initialize array length
	output := make([][]rune, app.ySize)
	for i := range output {
		output[i] = make([]rune, app.xSize)
		for j := range output[i] {
			switch {
			case i == app.applePos.y && j == app.applePos.x:
				output[i][j] = 'A'
			case i == app.ySize-1 || i == 0:
				output[i][j] = 'W'
			case j == app.xSize-1 || j == 0:
				output[i][j] = 'W'
			default:
				output[i][j] = ' '
			}
		}
	}

	// render snack on the map
	for _, pos := range snackPosition {
		conditionY_1 := pos.y >= 0
		conditionY_2 := pos.y < app.ySize
		conditionX_1 := pos.x >= 0
		conditionX_2 := pos.x < app.xSize
		if conditionY_1 && conditionY_2 && conditionX_1 && conditionX_2 {
			output[pos.y][pos.x] = 'O'
		}
	}
	app.screen.Clear()
	for i := range output {
		for j := range output[i] {
			app.screen.SetContent(j, i, output[i][j], nil, app.defStyle)
		}
	}
	app.screen.Show()
}

func (app *appConfig) quitGame() {
	close(app.gameExit)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
