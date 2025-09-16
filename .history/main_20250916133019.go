package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Player represents a connected player
type Player struct {
	Conn   *websocket.Conn
	Choice string
}

// Upgrader for HTTP → WebSocket upgrade
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins
}

var waitingPlayer *Player = nil // Player waiting for opponent

func playRPS(p1, p2 *Player) (string, string) {
	// Determine results
	c1 := p1.Choice
	c2 := p2.Choice

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

	// Wait for a choice from the player
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
		break
	}

	// Match with waiting player or wait for opponent
	if waitingPlayer == nil {
		waitingPlayer = player
		conn.WriteJSON(map[string]string{"status": "waiting for opponent"})
		return
	} else {
		opponent := waitingPlayer
		waitingPlayer = nil

		res1, res2 := playRPS(player, opponent)
		player.Conn.WriteJSON(map[string]string{"yourChoice": player.Choice, "opponentChoice": opponent.Choice, "result": res1})
		opponent.Conn.WriteJSON(map[string]string{"yourChoice": opponent.Choice, "opponentChoice": player.Choice, "result": res2})
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server running on ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
