package distro

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
)

type ricSender struct {
	client MQTT.Client
	topic  string
}

// For command info parse
type GetCommand struct {
	URL string
}
type Command struct {
	Get  GetCommand
	Name string
}
type Document struct {
	Commands []Command
}

// For event data parse
type Reading struct {
	Name  string
	Value string
}
type DataPacket struct {
	Device   string
	Origin   uint64
	Readings []Reading
}

// todo map object_id + commandsMap
var commandsMap = make(map[string]string)

// NewRicSender - create new mqtt sender for sandbox.rightech.io
func NewRicSender(newReg contract.Registration, cert string, key string) sender {
	addr := newReg.Addressable
	protocol := strings.ToLower(addr.Protocol)
	opts := MQTT.NewClientOptions()
	broker := protocol + "://" + addr.Address + ":" + strconv.Itoa(addr.Port) + addr.Path
	opts.AddBroker(broker)
	opts.SetClientID(addr.Publisher)
	opts.SetUsername(addr.User)
	opts.SetPassword(addr.Password)
	opts.SetAutoReconnect(false)

	if protocol == "tcps" || protocol == "ssl" || protocol == "tls" {
		cert, err := tls.LoadX509KeyPair(cert, key)

		if err != nil {
			LoggingClient.Error("Failed loading x509 data")
			return nil
		}

		tlsConfig := &tls.Config{
			ClientCAs:          nil,
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{cert},
		}

		opts.SetTLSConfig(tlsConfig)

	}

	sender := &ricSender{
		client: MQTT.NewClient(opts),
		topic:  addr.Topic,
	}

	sender.Subscribe(newReg.Filter.DeviceIDs)

	return sender
}

func (sender *ricSender) Send(data []byte, ctx context.Context) bool {
	if !sender.client.IsConnected() {
		LoggingClient.Info("Connecting to mqtt server")
		if token := sender.client.Connect(); token.Wait() && token.Error() != nil {
			LoggingClient.Error(fmt.Sprintf("Could not connect to mqtt server, drop event. Error: %s", token.Error().Error()))
			return false
		}
	}

	dataPacket := DataPacket{}
	ioData := bytes.NewReader(data)
	if err := json.NewDecoder(ioData).Decode(&dataPacket); err != nil {
		LoggingClient.Error(fmt.Sprintf("Could not parse json. Error: %s", err.Error()))
		return false
	}
	paramsMap := make(map[string]string)
	paramsMap["time"] = string(dataPacket.Origin)
	for _, s := range dataPacket.Readings {
		paramsMap[s.Name] = s.Value
	}

	resultPacket, err := json.Marshal(paramsMap)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return false
	}

	LoggingClient.Debug(fmt.Sprintf("SEND: %X", resultPacket))

	token := sender.client.Publish(dataPacket.Device, 0, false, resultPacket)
	token.Wait()
	if token.Error() != nil {
		LoggingClient.Error(token.Error().Error())
		return false
	} else {
		LoggingClient.Debug(fmt.Sprintf("Sent data: %X", data))
		return true
	}
}

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())

	clientHttp := &http.Client{}
	var req *http.Request

	url, ok := commandsMap[message.Topic()]
	if ok != true {
		return
	}

	if len(message.Payload()) == 0 {
		fmt.Println("GET")
		reqGet, errHttp := http.NewRequest(
			"GET", url, nil)
		if errHttp != nil {
			fmt.Println(errHttp)
			return
		}
		req = reqGet
	} else {
		fmt.Println("SET")
		reqPut, errHttp := http.NewRequest(
			"PUT", url, bytes.NewBuffer(message.Payload()))
		if errHttp != nil {
			fmt.Println(errHttp)
			return
		}
		req = reqPut
	}
	req.Header.Set("Content-Type", "application/json")

	resp, errHttp := clientHttp.Do(req)
	if errHttp != nil {
		fmt.Println(errHttp)
		return
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func (sender *ricSender) Subscribe(deviceIdentifiers []string) {
	LoggingClient.Info(fmt.Sprintf("Subscribe()"))
	if !sender.client.IsConnected() {
		LoggingClient.Info("Connecting to mqtt server")
		if token := sender.client.Connect(); token.Wait() && token.Error() != nil {
			LoggingClient.Error(fmt.Sprintf("Could not connect to mqtt server, drop event. Error: %s", token.Error().Error()))
			return
		}
	}

	for i := 0; i < len(deviceIdentifiers); i++ {
		fmt.Println(deviceIdentifiers[i])
		getCommandsUrl(deviceIdentifiers[i])
	}
	fmt.Println("SIZE = ", len(commandsMap))

	for k := range commandsMap {
		fmt.Println("SUBSCRIBE ON", k)
		if token := sender.client.Subscribe(k, 0, onMessageReceived); token.Error() != nil {
			LoggingClient.Warn("mqtt error: ", token.Error())
		}
	}
}

func getCommandsUrl(deviceId string) {
	LoggingClient.Info(fmt.Sprintf("getCommandsUrl()"))
	clientHttp := &http.Client{}
	req, errHttp := http.NewRequest(
		// If you run edgex locally, use command below
		// "GET", "http://localhost:48082/api/v1/device/name/"+deviceId, nil)

		// If you run edgex in docker environment, use command below
		"GET", "http://edgex-core-command:48082/api/v1/device/name/"+deviceId, nil)
	if errHttp != nil {
		fmt.Println(errHttp)
		return
	}

	resp, errHttp := clientHttp.Do(req)
	if errHttp != nil {
		fmt.Println(errHttp)
		return
	}
	defer resp.Body.Close()

	commands := Document{}
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		LoggingClient.Error(fmt.Sprintf("Could not parse json. Error: %s", err.Error()))
		return
	}

	for _, s := range commands.Commands {

		/* Comment if not localhost */
		//pos := strings.Index(s.Get.URL, ":48082" )
		//var url = "http://localhost" + s.Get.URL[pos:]

		/* Exchange active string */
		// docker environment
		commandsMap[deviceId+"/"+s.Name] = s.Get.URL

		// local environment
		//commandsMap[deviceId + "/" + s.Name] = url
	}

}
