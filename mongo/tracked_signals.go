package mongo

import (
	"fmt"
	"bittrexProj/user"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
	"time"
	"math/rand"
)

func GetSignalPerUser(userID string, objectID int64) (*user.TrackedSignal, error) {
	signalCol := MgoSession.DB(userID).C("signals")
	trackedSignal := &user.TrackedSignal{}
	err := signalCol.FindId(objectID).One(trackedSignal)
	return trackedSignal, err
}

func GetSignalsPerUser(userID string) ([]*user.TrackedSignal, error) {
	signalCol := MgoSession.DB(userID).C("signals")
	var trackedSignals []user.TrackedSignal
	var trackedSignalsP []*user.TrackedSignal
	query := bson.M{}
	err := signalCol.Find(query).All(&trackedSignals)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	for i := range trackedSignals {
		trackedSignalsP = append(trackedSignalsP, &trackedSignals[i])
	}

	//fmt.Println("||| GetSignalsPerUser len(trackedSignalsP) = ", len(trackedSignalsP))

	return trackedSignalsP, nil
}

func CleanEditableSignals(userID string) (error) {
	signalCol := MgoSession.DB(userID).C("signals")
	query := bson.M{"status": "редактируется"}
	err := signalCol.Remove(query)
	if err != nil {
		fmt.Println("||| CleanEditableSignals Remove err = ", err)
		if err == mgo.ErrNotFound {
			return err
		}
		return err
	}
	return nil
}

func UpsertSignalByID(userID string, signalID int64, trackedSignal user.TrackedSignal) (err error) {
	signalCol := MgoSession.DB(userID).C("signals")
	if trackedSignal.ObjectID == 0 {
		trackedSignal.ObjectID = time.Now().Unix() + rand.Int63()
	}
	if _, err = signalCol.UpsertId(signalID, trackedSignal); err != nil {
		fmt.Println("||| UpsertSignalByID UpsertId err = ", err)
		return err
	}
	return nil
}

func InsertSignalsPerUser(userID string, trackedSignals []*user.TrackedSignal) (err error) {
	signalCol := MgoSession.DB(userID).C("signals")
	return signalCol.Insert(trackedSignals)
}

func InsertSignalPerUser(userID string, trackedSignals []*user.TrackedSignal) (err error) {
	signalCol := MgoSession.DB(userID).C("signals")
	return signalCol.Insert(trackedSignals)
}

func DeleteSignal(userID string, signalID int64) (err error) {
	signalCol := MgoSession.DB(userID).C("signals")

	signalCol.RemoveId(signalID)
	return
}

//func InsertSignalsPerUser(userID string, signals []*user.TrackedSignal) (err error) {
//	signalCol := MgoSession.DB(userID).C("signals")
//	for _, sig := range signals {
//		var err error
//		//index := mgo.Index{
//		//	Key: []string{"id"},
//		//}
//		//if err = MgoSession.DB(userID).C("signals").EnsureIndex(index); err != nil {
//		//	if err = MgoSession.DB(userID).C("signals").DropIndexName(index.Name); err != nil {
//		//		log.Printf("||| InsertSignals: Could not drop index to user with id %s: %s\n", userID, err.Error())
//		//	}
//		//	if err = MgoSession.DB(userID).C("signals").EnsureIndex(index); err != nil {
//		//		log.Printf("||| InsertSignals: Could not ensure index to user with id %s: %s\n", userID, err.Error())
//		//	}
//		//}
//		_, err = signalCol.UpsertId(sig.ID, sig)
//		if err != nil {
//			log.Printf("||| InsertSignals: Could not insert signal to user with id %s: %s\n", userID, err.Error())
//			continue
//		}
//	}
//	return
//}
