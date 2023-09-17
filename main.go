package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	posRight = iota - 2
	posDown
	_
	posUp
	posLeft
)

type gameConf struct {
	size  int
	level int
	debug bool
}
type appConfig struct {
	speed         time.Duration
	conf          gameConf
	curDirection  int
	nextDirection int
	curSnackBody  []position
	applePos      position
	clearCmd      *exec.Cmd
	gameExit      chan bool
	wg            sync.WaitGroup
	random        *rand.Rand
	score         int
}
type position struct {
	x int
	y int
}

func main() {
	app := gameInit()
	app.wg.Add(2)
	go app.keyPressEventListener()
	go app.snackMove()
	app.wg.Wait()
	fmt.Println("GAME OVER")
}
func gameInit() *appConfig {
	app := &appConfig{}

	flag.IntVar(&app.conf.size, "size", 30, "size of the map")
	flag.IntVar(&app.conf.level, "level", 1, "game difficulty 1 to 10")
	flag.BoolVar(&app.conf.debug, "debug", false, "debug mode (through wall)")
	flag.Parse()

	app.speed = time.Millisecond * time.Duration(525-app.conf.level*50)
	app.random = rand.New(rand.NewSource(time.Now().UnixNano()))

	app.clearCmd = exec.Command("clear")
	app.clearCmd.Stdout = os.Stdout

	app.gameExit = make(chan bool)

	app.curDirection = posUp
	app.nextDirection = posUp

	app.curSnackBody = make([]position, 5)
	for i := range app.curSnackBody {
		app.curSnackBody[i].x = app.conf.size / 2
		app.curSnackBody[i].y = app.conf.size/2 - 2 + i
	}

	app.genNewApple()

	return app
}
func (app *appConfig) genNewApple() {
	app.applePos = position{
		x: rand.Intn(app.conf.size-2) + 1,
		y: rand.Intn(app.conf.size-2) + 1,
	}
}
func (app *appConfig) keyPressEventListener() {
	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer app.wg.Done()
	defer keyboard.Close()
	for {
		select {
		case <-app.gameExit:
			return
		default:
			_, key, err := keyboard.GetKey()
			if err != nil {
				log.Fatal(err)
			}
			switch key {
			case keyboard.KeyArrowUp:
				app.nextDirection = posUp
			case keyboard.KeyArrowDown:
				app.nextDirection = posDown
			case keyboard.KeyArrowLeft:
				app.nextDirection = posLeft
			case keyboard.KeyArrowRight:
				app.nextDirection = posRight
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
			for _, pos := range app.curSnackBody {
				if newSnack[0] == pos {
					app.quitGame()
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
				newSnack = append(newSnack, app.curSnackBody[len(app.curSnackBody)-1])
				app.genNewApple()
				app.score += 10
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
		newPosition.x = app.conf.size - 2
	case snackHead.x == app.conf.size-1:
		newPosition.x = 1
	case snackHead.y == 0:
		newPosition.y = app.conf.size - 2
	case snackHead.y == app.conf.size-1:
		newPosition.y = 1
	}
	if newPosition != snackHead {
		return newPosition, true
	}
	return snackHead, false
}
func (app *appConfig) render(snackPosition []position) {
	// initialize array length
	output := make([][]string, app.conf.size)
	for i := range output {
		output[i] = make([]string, app.conf.size)
		for j := range output[i] {
			switch {
			case i == app.applePos.y && j == app.applePos.x:
				output[i][j] = "A"
			case i == app.conf.size-1 || i == 0:
				output[i][j] = "W"
			case j == app.conf.size-1 || j == 0:
				output[i][j] = "W"
			default:
				output[i][j] = " "
			}
		}
	}

	// render snack on the map
	for _, pos := range snackPosition {
		conditionY_1 := pos.y >= 0
		conditionY_2 := pos.y < app.conf.size
		conditionX_1 := pos.x >= 0
		conditionX_2 := pos.x < app.conf.size
		if conditionY_1 && conditionY_2 && conditionX_1 && conditionX_2 {
			output[pos.y][pos.x] = "O"
		}
	}
	app.clearCmd.Run()
	fmt.Printf("SCORE : %d\n", app.score)
	for i := range output {
		for j := range output[i] {
			fmt.Print(output[i][j])
		}
		fmt.Print("\n")
	}
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
