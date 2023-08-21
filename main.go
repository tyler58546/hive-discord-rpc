package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/hugolgst/rich-go/client"
	"github.com/tyler58546/go-hive-api/hive"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	DiscordToken    = "1070830691757068389"
	Timeout         = time.Minute * 20
	RefreshInterval = 30 * time.Second
)

var Months = map[time.Month]string{
	1:  "JAN",
	2:  "FEB",
	3:  "MAR",
	4:  "APR",
	5:  "MAY",
	6:  "JUN",
	7:  "JUL",
	8:  "AUG",
	9:  "SEP",
	10: "OCT",
	11: "NOV",
	12: "DEC",
}

func main() {
	fmt.Printf("Hive Discord Rich Presence\n\n")
	for {
		fmt.Print("Enter username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _, _ := reader.ReadLine()
		player, err := hive.GetPlayer(string(username))
		if err != nil {
			if errors.Is(err, hive.ErrorInvalidPlayer) {
				fmt.Println("Invalid player. Please check your spelling and try again.")
				continue
			} else {
				log.Fatalln(err.Error())
			}
		}
		fmt.Printf("\nNow tracking %s.\n", player.UsernameCc)
		fmt.Printf("Rich presence will start after you finish your next game.\n\n")
		rpc := HiveDiscordRpc{player: player}
		rpc.Start()
		break
	}
}

type HiveDiscordRpc struct {
	player      *hive.Player
	currentGame *hive.Game
	startTime   time.Time
	lastUpdate  time.Time
}

func (r *HiveDiscordRpc) Start() {
	r.player.Handler = r
	_ = r.player.Update()
	t := time.NewTicker(RefreshInterval)
	for {
		<-t.C

		err := r.player.Update()
		if err != nil {
			log.Printf(err.Error())
		}

		if r.currentGame != nil {
			if !isMinecraftRunning() {
				log.Printf("Rich presence disabled because Minecraft was closed or suspended.")
				r.currentGame = nil
				client.Logout()
			} else if time.Now().Sub(r.lastUpdate) > Timeout {
				log.Printf("Rich presence disabled due to no recent activity.")
				r.currentGame = nil
				client.Logout()
			}
		}

	}
}

func (r *HiveDiscordRpc) HandleStatsUpdated(currentGame *hive.Game) {
	r.lastUpdate = time.Now()

	allTimeStats := r.player.AllTimeStatistics(currentGame)
	monthlyStats := r.player.MonthlyStatistics(currentGame)

	if r.currentGame != currentGame {
		r.startTime = time.Now()
		r.currentGame = currentGame
		log.Printf("Current game is now %s.", currentGame.Name)
	}

	var gameInfo []string
	if allTimePos, ok := r.player.AllTimeLeaderboardPosition(currentGame); ok {
		gameInfo = append(gameInfo, fmt.Sprintf("#%d All-Time [%d]", allTimePos, allTimeStats.GetInt(hive.StatisticWins)))
	} else {
		gameInfo = append(gameInfo, fmt.Sprintf("%d All-Time Wins", allTimeStats.GetInt(hive.StatisticWins)))
	}
	if monthlyPos, ok := r.player.MonthlyLeaderboardPosition(currentGame); ok {
		month := Months[time.Now().UTC().Month()]
		gameInfo = append(gameInfo, fmt.Sprintf("#%d %s [%d]", monthlyPos, month, monthlyStats.GetInt(hive.StatisticWins)))
	} else {
		if level := allTimeStats.GetInt(hive.StatisticLevel); level != 0 {
			gameInfo = append(gameInfo, fmt.Sprintf("| Level %d", level))
		} else {
			gameInfo = append(gameInfo, fmt.Sprintf("| %d XP", allTimeStats.GetInt(hive.StatisticExperience)))
		}
	}

	err := client.Login(DiscordToken)
	if err != nil {
		log.Printf("Failed to start Discord rich presence. Is Discord running?")
		return
	}

	err = client.SetActivity(client.Activity{
		Details:    currentGame.Name,
		State:      strings.Join(gameInfo, " "),
		LargeImage: currentGame.Id,
		Timestamps: &client.Timestamps{
			Start: &r.startTime,
		},
	})
	if err != nil {
		log.Printf(err.Error())
	}
}

func isMinecraftRunning() bool {
	out, _ := runPowershellCommand("((Get-Process -Name Minecraft.Windows -ErrorAction SilentlyContinue).Threads | Where-Object {$_.ThreadState -eq \"Wait\"}).Count -eq ((Get-Process -Name Minecraft.Windows -ErrorAction SilentlyContinue).Threads).Count")
	return out == "False\r\n"
}

func runPowershellCommand(cmd string) (string, error) {
	outBytes, err := exec.Command("powershell", "-Command", cmd).Output()
	return string(outBytes), err
}
