package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fenwickelliott/rome/model"
)

var (
	nodeID string
	port   string
)

var (
	leader      string
	state       model.State
	term        uint64
	heartBeatOK = make(chan struct{}, 1)
)

var (
	client = &http.Client{}
	peers  []string
)

func server() {
	r := gin.New()
	r.GET("/heartbeat", func(c *gin.Context) {
		t, err := strconv.ParseUint(c.Query("term"), 10, 64)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(400)
			return
		}

		if t < term {
			// log.Println("received stale heatbeat, dropping")
		} else if t == term {
			if leader != c.Query("leader") {
				term++
				state = model.Candidate
				leader = nodeID
			}
			select {
			case heartBeatOK <- struct{}{}:
			default:
			}
		} else if t > term {
			term = t
			leader = c.Query("leader")
			state = model.Follower
			log.Println("term incremented to ", term)
		} else {
			panic("imposible")
		}
	})
	r.GET("/vote", func(c *gin.Context) { // in case of new term suppoort candidate, other wise support leader
		t, err := strconv.ParseUint(c.Query("term"), 10, 64)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(400)
			return
		}
		if t > term {
			c.JSON(200, map[string]string{"leader": c.Query("called_by")})
		} else {
			c.JSON(200, map[string]string{"leader": leader})
		}
	})

	fatal(r.Run(port))
}

func stateEngine() {
	for {
	stateEngineSwitch:
		switch state {
		case model.Follower:
			electionTimeout := time.NewTimer(150 + time.Duration(rand.Intn(150))*time.Millisecond)
			for range electionTimeout.C {
				select {
				case <-heartBeatOK:
					// log.Println("heartBeatOK")
					electionTimeout.Reset(150 + time.Duration(rand.Intn(150))*time.Millisecond)
				default:
					log.Println("election timeout")
					state = model.Candidate
					break stateEngineSwitch
				}
			}
		case model.Candidate:
			term++
			votes := map[string]int{}
			votes[nodeID]++
			for _, p := range peers {
				values := url.Values{
					"term":      []string{strconv.FormatUint(term, 10)},
					"called_by": []string{nodeID},
				}
				resp, err := client.Get(p + "/vote?" + values.Encode())
				if err != nil {
					log.Println(err)
					continue
				}
				defer resp.Body.Close()

				var vote struct {
					Leader string
				}
				err = json.NewDecoder(resp.Body).Decode(&vote)
				if err != nil {
					log.Println(err)
					continue
				}

				votes[vote.Leader]++
				if votes[vote.Leader]*2 > len(peers) {
					log.Println("election won")
					state = model.Leader
					leader = nodeID
					break stateEngineSwitch
				}
			}
			log.Println("election lost")
			state = model.Follower
			break stateEngineSwitch
		case model.Leader:
			for _, p := range peers {
				values := url.Values{
					"term":   []string{strconv.FormatUint(term, 10)},
					"leader": []string{nodeID},
				}
				_, err := client.Get(p + "/heartbeat?" + values.Encode())
				if err != nil {
					// log.Println(err)
					continue
				}
			}
		}
	}
}

func main() {
	go server()
	go stateEngine()
	go func() {
		for range time.NewTicker(time.Second).C {
			log.Println("node state:", state, "term:", term, "leader:", leader)
		}
	}()

	log.Printf("rome-%s listening on %s", nodeID, port)
	select {}
}

func init() {
	flag.StringVar(&nodeID, "nodeID", "", "nodeID")
	flag.StringVar(&port, "p", "", "port")
	flag.Parse()

	if nodeID == "" {
		log.Fatal("no nodeID given")
	}

	if port == "" {
		log.Fatal("no port given")
	}
	port = ":" + port

	rand.Seed(time.Now().UnixNano())

	for _, p := range []string{"http://localhost:8000", "http://localhost:8001", "http://localhost:8002"} {
		if !strings.Contains(p, port) {
			peers = append(peers, p)
		}
	}

	gin.SetMode(gin.ReleaseMode)
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
