package main
 
import (
	"os"
	"bufio"
	"log"
	"net"
	"fmt"
	"time"
	"encoding/json"
	r "math/rand"
	"strings"
)

type SensorData struct {  
    Location   	string 	`json:"location"`
    Date 		string	`json:"date"`
    Input 		int		`json:"in"`
    Output		int	   	`json:"out"`
}

func main() {
	if (len(os.Args) != 2) {
		fmt.Println("Usage: ", os.Args[0], " <location>")
		os.Exit(1)
	}
	// connexion a un serveur en localhost sur le port 1111
	conn, err := net.Dial("tcp", "localhost:1111")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Connexion avec le serveur realisee")
	// logique client
	generateAndSendData(conn)
	// coupure de la connexion
	conn.Close()
	fmt.Println("Connexion avec le serveur coupee")
}

func generateAndSendData(conn net.Conn) {
	// creation d'un flux d'ecriture sur la connexion
	writer := bufio.NewWriter(conn)
	// creation d'une seed pour le random
	r.Seed(time.Now().UnixNano())

	// arrondi de la date pour avoir une uniformite des requetes cote serveur
	round := (10 * time.Second)

	// genere une instance de SensorData et la convertit en JSON
	data := SensorData{
		os.Args[1],
		time.Now().Truncate(round).Format(time.RFC3339),
		r.Intn(10),
		r.Intn(10),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}
	// boucle for, 3 essais ou response == "ACK"
	for i := 0; i < 4; i++ {
		// ecrit le JSON sur la connexion, le caractere '\n' delimite le message
		writer.WriteString(strings.TrimSpace(string(jsonData)) + "\n")
		writer.Flush()
		// gestion de la reponse du serveur
		response, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil || response != "ACK\n" {
			log.Println(err)
		} else {
			// flush du flux d'ecriture
			writer.Flush()
			break
		}
	}
}