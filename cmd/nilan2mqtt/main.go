package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/pjuzeliunas/nilan"
	"github.com/pjuzeliunas/nilan2mqtt/internal"
	"github.com/pjuzeliunas/nilan2mqtt/internal/config"
)

func mqttClient(brokerAddress string, port int, username string, password string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", brokerAddress, port))
	opts.SetClientID(uuid.New().String())
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		panic(err)
	}
	opts.SetDefaultPublishHandler(processMessage)
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return c
}

func subscribeForTopics(client mqtt.Client) {
	topics := []string{
		"homeassistant/fan/nilan/set",
		"homeassistant/fan/nilan/speed/set",
		"homeassistant/fan/nilan/mode/set",
	}
	for _, t := range topics {
		token := client.Subscribe(t, 1, nil)
		token.Wait()
	}
}

func processMessage(client mqtt.Client, msg mqtt.Message) {
	payload := string(msg.Payload())
	switch msg.Topic() {
	case "homeassistant/fan/nilan/set":
		settings := nilan.Settings{
			VentilationOnPause: boolAddr(payload == "OFF"),
		}
		NilanController.SendSettings(settings)
	case "homeassistant/fan/nilan/speed/set":
		speed, _ := strconv.Atoi(payload)
		settings := nilan.Settings{
			FanSpeed: internal.FanSpeed(speed),
		}
		NilanController.SendSettings(settings)
	case "homeassistant/fan/nilan/mode/set":
		settings := nilan.Settings{
			VentilationMode: internal.Mode(payload),
		}
		NilanController.SendSettings(settings)
	}
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func boolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func sendSimpleConfig(client mqtt.Client, topic string, config config.SimpleConfig) {
	d, _ := json.Marshal(config)
	t := client.Publish(topic, 0, false, d)
	t.Wait()
}

func sendFanConfig(client mqtt.Client, topic string, config config.Fan) {
	d, _ := json.Marshal(config)
	t := client.Publish(topic, 0, false, d)
	t.Wait()
}

func setUpConfig(client mqtt.Client) {
	sendSimpleConfig(client, "homeassistant/sensor/nilan/1/config", config.RoomTemperature())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/2/config", config.OutdoorTemperature())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/3/config", config.HumidityAvg())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/4/config", config.Humidity())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/5/config", config.DHWTemperatureTop())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/6/config", config.DHWTemperatureBottom())
	sendSimpleConfig(client, "homeassistant/sensor/nilan/7/config", config.SupplyFlowTemperature())
	sendFanConfig(client, "homeassistant/fan/nilan/config", config.NilanVentilation())
}

func publishReadings(client mqtt.Client, readings internal.ReadingsDTO) {
	d, _ := json.Marshal(readings)
	t := client.Publish("homeassistant/sensor/nilan/state", 0, false, d)
	t.Wait()
}

func publishVentilationState(client mqtt.Client, ventilationState internal.VentilationDTO) {
	d, _ := json.Marshal(ventilationState)
	t := client.Publish("homeassistant/fan/nilan/state", 0, false, d)
	t.Wait()
}

var NilanController nilan.Controller = nilan.Controller{Config: nilan.Config{NilanAddress: "192.168.1.31:502"}}

func main() {
	mqttC := mqttClient("192.168.1.18", 1883, "", "")
	defer mqttC.Disconnect(0)

	setUpConfig(mqttC)
	subscribeForTopics(mqttC)

	for {
		readings := NilanController.FetchReadings()
		readingsDTO := internal.CreateReadingsDTO(readings)
		publishReadings(mqttC, readingsDTO)

		settings := NilanController.FetchSettings()
		ventilationDTO := internal.CreateVentilationDTO(settings)
		publishVentilationState(mqttC, ventilationDTO)

		time.Sleep(5 * time.Second)
	}
}
