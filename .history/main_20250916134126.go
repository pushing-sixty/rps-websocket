package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Player struct {
	Conn   *websocket.Conn
	Choice string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var waitingPlayer *Player
var mu sync.Mutex // Protect access to waitingPlayer

func playRPS(p1, p2 *Player) (string, string) {
	c1, c2 := p1.Choice, p2.Choice
	result := func(a, b string) string {
		if a == b {
			return "draw"
		}
		if (a == "rock" && b == "scissors") ||
			(a == "paper" && b == "rock") ||
			(a == "scissors" && b == "paper") {
			return "win"
		}
		return "lose"
	}
	return result(c1, c2), result(c2, c1)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	player := &Player{Conn: conn}

	for {
		var msg struct {
			Choice string `json:"choice"`
		}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		player.Choice = msg.Choice

		mu.Lock()
		if waitingPlayer == nil {
			// No one waiting → this player waits
			waitingPlayer = player
			conn.WriteJSON(map[string]string{"status": "waiting for opponent"})
			mu.Unlock()
		} else {
			// Match found → compute results
			opponent := waitingPlayer
			waitingPlayer = nil
			mu.Unlock()

			res1, res2 := playRPS(player, opponent)

			player.Conn.WriteJSON(map[string]string{
				"yourChoice":     player.Choice,
				"opponentChoice": opponent.Choice,
				"result":         res1,
			})
			opponent.Conn.WriteJSON(map[string]string{
				"yourChoice":     opponent.Choice,
				"opponentChoice": player.Choice,
				"result":         res2,
			})
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server running on ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
