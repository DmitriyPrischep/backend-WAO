package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/spf13/viper"
)

var WidthField float64 = viper.GetFloat64("canvas.widthField")
var HeightField float64 = viper.GetFloat64("canvas.heightField")

var maxScrollHeight float64 = 0.25 * HeightField
var minScrollHeight float64 = 0.75 * HeightField

var randomGame *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano())) // Randomizer initialize
var gravity float64 = viper.GetFloat64("settings.gravity")
var spacing float64 = viper.GetFloat64("settings.spacing")

var koefScrollSpeed float64 = viper.GetFloat64("settings.koefScrollSpeed") // Скорость с которой все объекты будут падать вниз
// Настройки генерации карты
var koefGeneratePlates float64 = viper.GetFloat64("settings.koefGeneratePlates")
var koefHeightOfMaxGenerateSlice int = viper.GetInt("settings.koefHeightOfMaxGenerateSlice")

var leftIndent float64 = viper.GetFloat64("settings.leftIndent")
var rightIndent float64 = viper.GetFloat64("settings.rightIndent")

func CheckDevelopmentEnvironment() bool {
	if viper.ConfigFileUsed() != "./config/test.yml" {
		return true
	}
	return false
}

func FieldGenerator(beginY float64, b float64, k uint16) (newBlocks []*Block) {
	var WidthField float64 = viper.GetFloat64("canvas.widthField")
	var leftIndent float64 = viper.GetFloat64("settings.leftIndent")
	var rightIndent float64 = viper.GetFloat64("settings.rightIndent")
	// beginY was sended as the parameter
	p := b / float64(k) // Плотность
	var currentX float64
	currentY := beginY
	var i uint16
	for i = 0; i < k; i++ {
		currentX = randomGame.Float64()*((WidthField-rightIndent)-leftIndent+1) + leftIndent
		newBlocks = append(newBlocks, &Block{
			X:  currentX,
			Y:  currentY,
			Dy: 0,
			w:  viper.GetFloat64("block.width"),
			h:  viper.GetFloat64("block.height"),
		})
		currentY -= p
	}
	return
}

// Функция изменения скорости

func ProcessSpeed(delay float64, player *Player, gravity float64) {
	player.Dy += (gravity * delay)
}

// Отрисовка по кругу
func CircleDraw(player *Player) {
	WidthField := viper.GetFloat64("canvas.widthField")
	if player.X > WidthField {
		player.X = 0
	} else if player.X < 0 {
		player.X = WidthField
	}
}

func KillPlayer(player *Player) {
	player.room.Players.Range(func(_, player interface{}) bool {
		idPlayer, err := json.Marshal(player.(*Player).IdP)
		if err != nil {
			log.Println("Error with encoding player's id was occured", err)
			return true
		}
		player.(*Player).SendMessage(&Message{
			Type:    "lose",
			Payload: idPlayer,
		})
		return true
	})
}

func Collision(delay float64, player *Player, plate *Block) {
	if plate == nil {
		log.Printf("* Plate is nil * for player id%d", player.IdP)
		return
	}
	if player.Dy >= 0 {
		if player.Y+player.Dy*delay < plate.Y-plate.h {
			return
		}
		player.Y = plate.Y - plate.h
		player.Jump()
	}
}

func (player *Player) BlocksToAnotherCanvas(blocks []*Block, b float64) []*Block {
	var newBlocks []*Block
	for _, block := range blocks {
		blockCopy := *block
		blockCopy.Y = blockCopy.Y - (blocks[0].Y - b) + player.canvas.y
		newBlocks = append(newBlocks, &blockCopy)
	}
	return newBlocks
}

// Virtual transfer player to anotherPlayer's canvas
func (player *Player) playerToAnotherCanvas(anotherPlayer *Player) *Player {
	playerCopy := *player
	playerCopy.Y += (anotherPlayer.canvas.y - player.canvas.y)
	return &playerCopy
}

func (room *Room) AllPlayersToAnotherCanvas(player *Player) []*Player {
	var players []*Player
	room.Players.Range(func(_, plr interface{}) bool { // plr - a current player
		players = append(players, plr.(*Player).playerToAnotherCanvas(player))
		return true
	})
	return players
}

func (room *Room) HighestPlayer() *Player {
	var maxYPlayer *Player = nil
	maxY := math.MaxFloat64
	room.mutexEngine.Lock()
	room.Players.Range(func(_, player interface{}) bool {
		if player.(*Player).Y < maxY {
			maxY = player.(*Player).Y
			maxYPlayer = player.(*Player)
		}
		return true
	})
	room.mutexEngine.Unlock()
	return maxYPlayer
}

func (player *Player) Move(command *Command) {
	if command.Direction == "LEFT" {
		player.X -= player.Dx * command.Delay
	} else if command.Direction == "RIGHT" {
		player.X += player.Dx * command.Delay
	}
}

func (player *Player) PlayerMoveWithGravity(delay float64) {
	player.Y += (player.Dy * delay)
}

func (canvas *Canvas) CanvasMove(delay float64) {
	canvas.y += canvas.dy * delay
}

func (player *Player) MapUpdate(lastBlock *Block) (newBlocks []*Block, b float64) {
	beginY := lastBlock.Y - viper.GetFloat64("settings.spacing")
	koefHeightOfMaxGenerateSlice := viper.GetInt("settings.koefHeightOfMaxGenerateSlice")
	koefGeneratePlates := viper.GetFloat64("settings.koefGeneratePlates")
	b = float64(koefHeightOfMaxGenerateSlice) + (lastBlock.Y - player.canvas.y)
	k := uint16(koefGeneratePlates * (float64(koefHeightOfMaxGenerateSlice) + (lastBlock.Y - player.canvas.y)))
	newBlocks = FieldGenerator(beginY, b, k)
	return
}

func (player *Player) StartScrolling() {
	player.stateScrollMap = true // Сигнал запрещающий выполнять этот код еще раз пока не выполнится else
	player.canvas.dy = -viper.GetFloat64("settings.koefScrollSpeed")
}

func (player *Player) StopScrolling() {
	player.stateScrollMap = false // Scrolling was finished
	player.canvas.dy = 0
}

func Engine(player *Player) {
	defer func() {
		if e := recover(); e != nil {
			log.Println("Error at physic treatment was occured (function Engine)", e)
			if CheckDevelopmentEnvironment() {
				KillPlayer(player)
			}
			RemovePlayer(player)
		}
	}()
	var maxCountOfCommands uint64 = viper.GetUint64("player.maxCountOfCommands")
	var HeightField float64 = viper.GetFloat64("canvas.heightField")
	var gravity float64 = viper.GetFloat64("settings.gravity")
	var maxScrollHeight float64 = 0.25 * HeightField
	var minScrollHeight float64 = 0.75 * HeightField
	for {
		select {
		case <-player.engineDone:
			return
		default:
			if player.Y-player.H > player.canvas.y+HeightField {
				// log.Printf("Player with id %d lose!\n", player.IdP)
				if CheckDevelopmentEnvironment() {
					KillPlayer(player)
				}
				RemovePlayer(player)
				return
			}
			if player.Y-player.canvas.y <= maxScrollHeight && player.stateScrollMap == false {
				player.room.mutexEngine.Lock()
				player.StartScrolling()
				player.room.mutexEngine.Unlock()
				log.Printf("Canvas with player id%d is moving...\n", player.IdP)
				if player == player.room.HighestPlayer() {
					player.room.mutexEngine.Lock()
					player.room.scrollCount++
					player.room.mutexEngine.Unlock()
					player.room.mutexEngine.Lock()
					lastBlock := player.room.Blocks[len(player.room.Blocks)-1]
					player.room.mutexEngine.Unlock()
					newBlocks, b := player.MapUpdate(lastBlock)
					player.room.mutexEngine.Lock()
					player.room.Blocks = append(player.room.Blocks, newBlocks...)
					player.room.mutexEngine.Unlock()
					var buffer []byte
					var err error
					player.room.mutexEngine.Lock()
					player.room.Players.Range(func(_, playerWithCanvas interface{}) bool {
						var players []*Player
						player.room.Players.Range(func(_, player interface{}) bool {
							playerCopy := player.(*Player).playerToAnotherCanvas(playerWithCanvas.(*Player))
							players = append(players, playerCopy)
							return true
						})
						newBlocksForPlayer := playerWithCanvas.(*Player).BlocksToAnotherCanvas(newBlocks, b)
						if buffer, err = json.Marshal(struct {
							Blocks  []*Block  `json:"blocks"`
							Players []*Player `json:"players"`
						}{
							Blocks:  newBlocksForPlayer,
							Players: players,
						}); err != nil {
							fmt.Println("Error encoding new blocks", err)
							player.room.mutexEngine.Unlock()
							return false // ?
						}
						if CheckDevelopmentEnvironment() {
							playerWithCanvas.(*Player).SendMessage(&Message{
								Type:    "map",
								Payload: buffer,
							})
						}
						return true
					})
					player.room.mutexEngine.Unlock()
					log.Println("******* MAP WAS SENDED *******")
					log.Println("New blocks:")
					for _, block := range newBlocks {
						fmt.Printf("x: %f, y: %f, w: %f, h: %f\n", block.X, block.Y, block.w, block.h)
					}

				}
			} else if player.Y-player.canvas.y >= minScrollHeight && player.stateScrollMap == true {
				player.room.mutexEngine.Lock()
				player.StopScrolling()
				player.room.mutexEngine.Unlock()
				log.Printf("Canvas with player id%d was stopped...\n", player.IdP)
			}
			player.room.mutexEngine.Lock()
			CircleDraw(player)
			player.room.mutexEngine.Unlock()
			select {
			case command := <-player.commands:
				if command == nil {
					fmt.Println("Command's error was occured")
					continue
				}
				if player.commandCounter == maxCountOfCommands {
					player.room.mutexEngine.Lock()
					players := player.room.AllPlayersToAnotherCanvas(player)
					buf, err := json.Marshal(players)
					if err != nil {
						log.Println("Error players to encoding")
						player.room.mutexEngine.Unlock()
						continue
					}
					if CheckDevelopmentEnvironment() {
						player.SendMessage(&Message{
							Type:    "updatePositions",
							Payload: buf,
						})
					}
					player.commandCounter = 0
					player.room.mutexEngine.Unlock()
				} else {
					player.room.mutexEngine.Lock()
					player.commandCounter++
					player.room.mutexEngine.Unlock()
				}
				if command.Direction == "LEFT" || command.Direction == "RIGHT" {
					player.room.mutexEngine.Lock()
					player.Move(command)
					player.room.mutexEngine.Unlock()
				}
				player.room.mutexEngine.Lock()
				ProcessSpeed(command.Delay, player, gravity)
				Collision(command.Delay, player, player.SelectNearestBlock(&player.room.Blocks))
				player.PlayerMoveWithGravity(command.Delay)
				player.canvas.CanvasMove(command.Delay)
				player.room.mutexEngine.Unlock()
			}
		}
	}
}
