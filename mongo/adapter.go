package mongo

import (
	"gopkg.in/mgo.v2"
	"log"
	"fmt"
	"bittrexProj/user"
)

const (
	SERVER_MGO_IP   = "206.189.66.4" // ip DO
	SERVER_MGO_PORT = "27017"
)

var MgoSession *mgo.Session

func Connect() {
	MgoSession = getSession()
	if MgoSession == nil {
		fmt.Println("||| Connect: mongoSession is nil")
		return
	} else {
		fmt.Println("||| Connect: mongoSession not equal nil")
		MgoSession.SetMode(mgo.Monotonic, true)
	}

	user.GetSignalPerUserLink = GetSignalPerUser
	user.GetSignalsPerUserLink = GetSignalsPerUser
	user.UpsertSignalByIDLink = UpsertSignalByID
	user.DeleteSignalLink = DeleteSignal
	user.InsertSignalsPerUserLink = InsertSignalsPerUser
	user.OneLink = One
	user.UpsertUserByIDLink = UpsertUserByID
}

func getSession() *mgo.Session {
	session, err := mgo.Dial(SERVER_MGO_IP + ":" + SERVER_MGO_PORT)
	if err != nil {
		log.Printf("||| getSession: Could not connect to mongo: %s\n", err.Error())
		return nil
	}
	return session
}
