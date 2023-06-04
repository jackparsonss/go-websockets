package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsRequest)

var clients = make(map[WebSocketConnection]string)

var views = jet.NewSet(jet.NewOSFileSystemLoader("./html"), jet.InDevelopmentMode())

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Home(w http.ResponseWriter, r *http.Request) {
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		fmt.Printf("failed to get home page: %v\n", err)
	}
}

type WebSocketConnection struct {
	*websocket.Conn
}

type WsResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	Messagetype    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

type WsRequest struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("failed to get upgrade to websocket connection: %v\n", err)
	}

	log.Println("client connected to websocket endpoint")

	payload := WsRequest{
		Message: "connected to server",
		Action:  "USER_ENTERED",
	}

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	wsChan <- payload
	go listenForWs(&conn)
}

func listenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("error: %v\n", r)
		}
	}()

	var payload WsRequest
	for {
		err := conn.ReadJSON(&payload)
		if err == nil {
			// no error
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChan() {
	var response WsResponse

	for {
		e := <-wsChan

		switch e.Action {
		case "USERNAME":
			clients[e.Conn] = e.Username

			users := getUserList()
			response.ConnectedUsers = users
			response.Action = "LIST_USERS"
		case "USER_LEFT":
			delete(clients, e.Conn)

			users := getUserList()
			response.Action = "LIST_USERS"
			response.ConnectedUsers = users
		case "USER_ENTERED":
			users := getUserList()
			response.Action = "LIST_USERS"
			response.ConnectedUsers = users
		case "SEND_MESSAGE":
			response.Action = "SEND_MESSAGE"
			response.Message = fmt.Sprintf("%s: %s", e.Username, e.Message)
		}
		broadcastToAll(response)
	}
}

func getUserList() []string {
	var userList []string
	for _, user := range clients {
		if user != "" {
			userList = append(userList, user)
		}
	}

	sort.Strings(userList)
	return userList
}

func broadcastToAll(response WsResponse) {
	for client := range clients {
		err := client.WriteJSON(response)

		if err != nil {
			log.Printf("WS ERROR: %v", err)
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		return fmt.Errorf("failed to get template: %v", err)
	}

	err = view.Execute(w, data, nil)
	if err != nil {
		return fmt.Errorf("failed to display template: %v", err)
	}

	return nil
}
