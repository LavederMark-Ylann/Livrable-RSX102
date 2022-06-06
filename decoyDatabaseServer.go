package main

import (
    "fmt"
    "net"
    "bufio"
    "log"
)

func main() {
    // ecoute sur le port 1112
    listener, err := net.Listen("tcp", "localhost:1112")
    if err != nil {
        log.Fatalln(err)
    }
    fmt.Println("Serveur demarre sur le port 1112")
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
    // ouvre un flux de lecture sur la connexion
    reader := bufio.NewReader(conn)
    for {
        // lecture d'un message jusqu'a la rencontre du caractere '\n'
        netData, err := reader.ReadString('\n')
        if err != nil {
            log.Println(err)
            return
        }
        // affiche le message
        fmt.Println(netData)
    }
    conn.Close()
}