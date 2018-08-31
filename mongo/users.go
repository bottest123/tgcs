package mongo

import (
	"fmt"
	"bittrexProj/user"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
)

func UpsertUserByID(userID string, user user.User) (err error) {
	usersCol := MgoSession.DB("bot").C("users")
	if userID == "" {
		return
	}
	// user.ObjectID == "" возможно, если пользователь новый:
	if user.ObjectID == "" {
		user.ObjectID = userID
	}
	if _, err = usersCol.UpsertId(userID, user); err != nil {
		fmt.Printf("||| UpsertUserByID UpsertId userID = %s user.ObjectID = %v err = %v\n", userID, user.ObjectID, err)
		return err
	}
	return nil
}

func DeleteSoft(userID string) (err error) {
	usersCol := MgoSession.DB("bot").C("users")
	selector := bson.M{"_id": userID}
	update := bson.M{"$set": bson.M{"is_delete": true}}

	if err = usersCol.Update(selector, update); err != nil {
		fmt.Println("||| DeleteSoft Update err = ", err)
		return err
	}
	return nil
}

func DeleteUserHard(userID string) (err error) {
	usersCol := MgoSession.DB("bot").C("users")
	deletedUsersCol := MgoSession.DB("bot").C("deleted_users")

	user, err := One(userID)
	if err != nil {
		fmt.Println("||| DeleteUserHard One err = ", err)
		return err
	}
	if _, err = deletedUsersCol.UpsertId(userID, user); err != nil {
		fmt.Println("||| DeleteUserHard UpsertId err = ", err)
		return err
	}

	if err = usersCol.RemoveId(userID); err != nil {
		fmt.Println("||| DeleteUserHard RemoveId err = ", err)
		return err
	}

	return nil
}

func UpdateUser(userID interface{}, update bson.M) (err error) {
	usersCol := MgoSession.DB("bot").C("users")
	selector := bson.M{"_id": userID}
	return usersCol.Update(selector, update)
}

func All() ([]user.User, error) {
	usersCol := MgoSession.DB("bot").C("users")
	var users []user.User
	query := bson.M{} // "is_delete": bson.M{"$ne": true}
	err := usersCol.Find(query).All(&users)
	if err != nil {
		if err == mgo.ErrNotFound {
			return users, err
		}
		return nil, err
	}
	for _, us := range users {
		user.UserPropMap[string(us.ObjectID)] = us
	}
	fmt.Println("||| All len(user.UserPropMap) = ", len(user.UserPropMap))

	return users, err
}

func One(userID string) (*user.User, error) {
	usersCol := MgoSession.DB("bot").C("users")
	user := &user.User{}
	err := usersCol.FindId(userID).One(user)
	return user, err
}
