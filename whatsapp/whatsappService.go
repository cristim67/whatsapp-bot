package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type Response struct {
	Status  string  `json:"status"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	City    string  `json:"city"`
}

type WhatsAppService struct {
	store *sqlstore.Container
}

func New() WhatsAppService {
	database_url := os.Getenv("WHATSAPP_POLL_DATABASE_URL")

	store, err := sqlstore.New("postgres", database_url, nil)
	if err != nil {
		panic(err)
	}

	return WhatsAppService{
		store: store,
	}
}

func (b WhatsAppService) Login() (string, error) {
	device, err := b.store.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(device, nil)
	if client.Store.ID != nil {
		b.Logout()
	}

	qrChan, _ := client.GetQRChannel(context.Background())
	err = client.Connect()
	if err != nil {
		panic(err)
	}
	defer client.Disconnect()

	for evt := range qrChan {
		if evt.Event == "code" {
			// Render the QR code to the terminal
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			return evt.Code, nil
		}
	}

	return "", nil
}

func (b WhatsAppService) Logout() error {
	devices, err := b.store.GetAllDevices()
	if err != nil {
		panic(err)
	}

	for _, device := range devices {
		b.store.DeleteDevice(device)
	}

	return nil
}

func (b WhatsAppService) ListenForMessages(seconds int) error {
	device, err := b.store.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(device, nil)
	if client.Store.ID == nil {
		return errors.New("client is not logged in")
	}

	client.AddEventHandler(func(evt interface{}) {
		switch evt := evt.(type) {
		case *events.Connected:
			client.SendPresence(types.PresenceAvailable)
			fmt.Println("Connected")
		case *events.Message:
			fmt.Printf("Message received in chat %s, from %s (%s) at %s\n", evt.Info.MessageSource.Chat.String(), evt.Info.MessageSource.Sender.String(), evt.Info.PushName, evt.Info.Timestamp)
		}
	})

	err = client.Connect()
	if err != nil {
		panic(err)
	}
	defer client.Disconnect()

	time.Sleep(time.Duration(seconds) * time.Second)

	return nil
}

func (b WhatsAppService) CreatePoll() error {
	device, err := b.store.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(device, nil)

	if client.Store.ID == nil {
		// Not logged in
		fmt.Println("Action did not run because the client is not logged in")
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		defer client.Disconnect()

		genezio_jid := "120363028452547709"
		poll := client.BuildPollCreation("Cand vii la birou?", []string{"9:00", "10:00", "11:00", "12:00", "13:00", "WFH", "OOO"}, 2)
		client.SendMessage(context.Background(), types.NewJID(genezio_jid, types.GroupServer), poll)

		fmt.Println("Created a poll as ", client.Store.PushName)
	}

	return nil
}
