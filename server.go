package main
 
import (
	"bufio"
	"log"
	"net"
	"strings"
	"fmt"
	"encoding/json"
	"time"
	"regexp"
)

type SensorData struct {  
    Location   	string 	`json:"location"`
    Date 		string	`json:"date"`
    Input 		int		`json:"in"`
    Output		int	   	`json:"out"`
}

type Key struct {
	Location string
	Date string
}

type Value struct {
	Input int
	Output int
}


type TrueValue struct {
	Location string
	Input int
	Output int
}

var datatable = []SensorData{}
var tempTable = []SensorData{}
var datatableIsUsed = false

func main() {
	// ecoute sur le port 1111
	listener, err := net.Listen("tcp", "0.0.0.0:1111")

	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Serveur demarre sur le port 1111")
	// connection au decoyDatabaseServer sur un autre thread
	go handleDatabaseRequest()
 
	for {
		// attente de connexion
		con, err := listener.Accept()
		if err != nil {
			log.Println(err)
		}
		// creation d'une goroutine pour chaque connexion, equivalent a un thread
		go handleClientRequest(con)
	}
}

func handleClientRequest(conn net.Conn) {
	fmt.Printf("Connection de %s\n", conn.RemoteAddr().String())
	// 4 essais max pour la reception des donnees
	for i := 0; i < 4; i++ {
		// lecture d'un message jusqu'a la rencontre du caractere '\n'
		netData, err := bufio.NewReader(conn).ReadString('\n')
		writer := bufio.NewWriter(conn)
		if err != nil {
			fmt.Println(err)
			writer.WriteString("NACK\n")
			writer.Flush()
		} else {
			writer.WriteString("ACK\n")
			writer.Flush()
			temp := strings.TrimSpace(string(netData))
			fmt.Println(conn.RemoteAddr().String() + " : " + temp)
			// Deserialise le netData en SensorData
			var data SensorData
			err = json.Unmarshal([]byte(temp), &data)
			if err != nil {
				log.Println(err)
			}
			// Ajoute le SensorData a la table de donnees si datatableIsUsed est faux, sinon, ajoute le SensorData a la table de donnees temporaire
			if datatableIsUsed == false {
				datatable = append(datatable, data)
			}
			if datatableIsUsed == true {
				tempTable = append(tempTable, data)
			}
			conn.Close()
			break
		}
	}
}

func handleDatabaseRequest() {
	// connexion a un serveur en localhost sur le port 1111
	conn, err := net.Dial("tcp", "localhost:1112")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Connecte au serveur de base de donnees")
	// logique client, monothread
	formatAndSendDataReceived(conn)
	// coupure de la connexion
	conn.Close()
} 

func formatAndSendDataReceived(conn net.Conn) {
	// creation d'un flux d'ecriture sur la connexion
	writer := bufio.NewWriter(conn)

	// arrondi de la date pour avoir le round cote client x2
	round := (20 * time.Second)

	// boucle while true
	for {
		// si la table de donnes n'est pas vide
		if len(datatable) != 0 {
			datatableIsUsed = true

			// somme des in/out et mappage par location et date
			// [location, date] => [in, out]
			summary := make(map[Key]Value)
			// creation d'un tableau de Key vide
			keysList := []Key{}
			for _, sensorData := range datatable {
				// supprime tous les chiffres dans la location
				reg, err := regexp.Compile("[^a-zA-Z]+")
				if err != nil {
					log.Fatal(err)
				}
				processedLocation := reg.ReplaceAllString(sensorData.Location, "")
				key := Key{processedLocation, sensorData.Date}
				if _, ok := summary[key]; ok {
					newValue := summary[key]
					newValue.Input += sensorData.Input
					newValue.Output += sensorData.Output
					summary[key] = newValue
				} else {
					// ajoute la Key a la liste des Key si elle n'existe pas, pour iterer dessus ulterieurement
					keysList = append(keysList, key)
					summary[key] = Value{sensorData.Input, sensorData.Output}
				}
			}

			// recapitulatif des donnees par date avec la location et la somme des entrees/sorties
			// date => {[location, in, out], [location, in, out], ...}
			trueSummary := make(map[string][]TrueValue)
			trueKeysList := []string{}
			for _, key := range keysList {
				trueKey := key.Date
				tempKey := Key{key.Location, key.Date}
				tempValue := summary[tempKey]
				trueValue := TrueValue{key.Location, tempValue.Input, tempValue.Output}
				if _, ok := trueSummary[trueKey]; !ok {
					// ajoute la TrueKey a la liste des TrueKey si elle n'existe pas, pour iterer dessus ulterieurement
					trueKeysList = append(trueKeysList, trueKey)
				}
				trueSummary[trueKey] = append(trueSummary[trueKey], trueValue)
			}
			
			for _, key := range trueKeysList {
				// format des donnees desirees par la base de donnees : 
				// 15:12:2107:10:10/Park:IN:20/Park:OUT:10/Center:IN:50/Center:OUT:30/Entree:IN:30/Caisse:OUT:30
				value := trueSummary[key]
				date, _ := time.Parse(time.RFC3339, key)
				formattedString := fmt.Sprintf("%02d:%02d:%02d%02d:%02d:%02d",
				date.Day(), date.Month(), date.Year() % 100,
				date.Hour(), date.Minute(), date.Second())
				for _, v := range value {
					if (v.Input > 0 && v.Output > 0) {
						formattedString += fmt.Sprintf("/%s:IN:%d/%s:OUT:%d", v.Location, v.Input, v.Location, v.Output)
					}
					if (v.Input > 0 && v.Output == 0) {
						formattedString += fmt.Sprintf("/%s:IN:%d", v.Location, v.Input)
					}
					if (v.Input == 0 && v.Output > 0) {
						formattedString += fmt.Sprintf("/%s:OUT:%d", v.Location, v.Output)
					}
				}

				// ecrit le resultat sur la connexion
				writer.WriteString(formattedString + "\n")
				// flush du flux d'ecriture
				writer.Flush()

			}
			// flush de la table de donnees
			datatable = []SensorData{}
			// ajoute la table de donnees temporaire dans la table de donnees
			datatable = append(datatable, tempTable...)
			// vide la table de donnees temporaire
			datatableIsUsed = false
			tempTable = []SensorData{}
		}
		// attend ROUND secondes
		time.Sleep(round)
	}
}