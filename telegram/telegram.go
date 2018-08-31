package telegram

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"bittrexProj/user"
	"bittrexProj/analizator"
	"github.com/shelomentsevd/mtproto"
	"io/ioutil"
	"encoding/json"

	"bittrexProj/config"
	"reflect"
	"bittrexProj/cryptoSignal"
	"bittrexProj/mongo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"bittrexProj/tools"
	"bittrexProj/smiles"
)

// based on github.com/shelomentsevd/telegramgo

const telegramAddress = "149.154.167.50:443"
const updatePeriod = time.Second * 2

var (
	//JoinStatusCHNameMap = map[string]string{}
	stopInit           = make(chan struct{}, 1)
	CliGlobal          *TelegramCLI
	BittrexBTCCoinList = map[string]bool{}
	mesGlobalMap       = map[string]bool{}
	approvedChans      = []string{}
	isStop             bool
	// для мониторинга = если запущен CLI - отправка через бот работает
	//CLIworks bool
)

type Command struct {
	Name      string
	Arguments string
}

// Returns user nickname in two formats:
// <id> <First name> @<Username> <Last name> if user has username
// <id> <First name> <Last name> otherwise
func nickname(user mtproto.TL_user) string {
	if user.Username == "" {
		return fmt.Sprintf("%d %s %s", user.Id, user.First_name, user.Last_name)
	}

	return fmt.Sprintf("%d %s @%s %s", user.Id, user.First_name, user.Username, user.Last_name)
}

// Returns date in RFC822 format
func formatDate(date int32) string {
	unixTime := time.Unix((int64)(date), 0)
	return unixTime.Format(time.RFC822)
}

// Reads user input and returns Command pointer
func (cli *TelegramCLI) readCommand() *Command {
	//fmt.Println("||| readCommand ")
	fmt.Printf("\nUser input: ")
	input, err := cli.reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if input[0] != '\\' {
		return nil
	}
	command := new(Command)
	input = strings.TrimSpace(input)
	args := strings.SplitN(input, " ", 2)
	command.Name = strings.ToLower(strings.Replace(args[0], "\\", "", 1))
	if len(args) > 1 {
		command.Arguments = args[1]
	}
	return command
}

// Show help
func help() {
	fmt.Println("Available commands:")
	fmt.Println("\\me - Shows information about current account")
	fmt.Println("\\contacts - Shows contacts list")
	fmt.Println("\\umsg <id> <message> - Sends message to user with <id>")
	fmt.Println("\\cmsg <id> <message> - Sends message to chat with <id>")
	fmt.Println("\\help - Shows this message")
	fmt.Println("\\quit - Quit")
}

type TelegramCLI struct {
	mtproto   *mtproto.MTProto
	state     *mtproto.TL_updates_state
	read      chan struct{}
	stop      chan struct{}
	connected bool
	reader    *bufio.Reader
	users     map[int32]mtproto.TL_user
	chats     map[int32]mtproto.TL_chat
	channels  map[int32]mtproto.TL_channel
}

func NewTelegramCLI(pMTProto *mtproto.MTProto) (*TelegramCLI, error) {
	if pMTProto == nil {
		return nil, errors.New("NewTelegramCLI: pMTProto is nil")
	}
	cli := new(TelegramCLI)
	cli.mtproto = pMTProto
	cli.read = make(chan struct{}, 1)
	cli.stop = make(chan struct{}, 1)
	cli.reader = bufio.NewReader(os.Stdin)
	cli.users = make(map[int32]mtproto.TL_user)
	cli.chats = make(map[int32]mtproto.TL_chat)
	cli.channels = make(map[int32]mtproto.TL_channel)

	return cli, nil
}

func (cli *TelegramCLI) Authorization(phonenumber string) error {
	if phonenumber == "" {
		return fmt.Errorf("Phone number is empty")
	}
	sentCode, err := cli.mtproto.AuthSendCode(phonenumber)
	if err != nil {
		return err
	}

	if !sentCode.Phone_registered {
		return fmt.Errorf("Phone number isn't registered")
	}

	var code string
	fmt.Printf("Enter code: ")
	fmt.Scanf("%s", &code)
	auth, err := cli.mtproto.AuthSignIn(phonenumber, code, sentCode.Phone_code_hash)
	if err != nil {
		return err
	}

	userSelf := auth.User.(mtproto.TL_user)
	cli.users[userSelf.Id] = userSelf
	message := fmt.Sprintf("Signed in: Id %d name <%s @%s %s>\n", userSelf.Id, userSelf.First_name, userSelf.Username, userSelf.Last_name)
	fmt.Print(message)
	log.Println(message)
	log.Println(userSelf)

	return nil
}

// Load contacts to users map
func (cli *TelegramCLI) LoadContacts() error {
	tl, err := cli.mtproto.ContactsGetContacts("")
	if err != nil {
		return err
	}
	list, ok := (*tl).(mtproto.TL_contacts_contacts)
	if !ok {
		return fmt.Errorf("RPC: %#v", tl)
	}
	for _, v := range list.Users {
		if v, ok := v.(mtproto.TL_user); ok {
			cli.users[v.Id] = v
		}
	}
	return nil
}

// Load contacts to users map
func (cli *TelegramCLI) LoadChannels() error {
	//inputchannel := mtproto.TL_inputChannel{}
	// John Galt & Phönix Gruppe(chanID: 1067657440 chanHash: 5092174803016167311)
	inputchannel := mtproto.TL_inputChannel{Channel_id: 1067657440, Access_hash: 5092174803016167311}
	ids := []mtproto.TL{inputchannel}

	//
	//list, ok := (*tl).(mtproto.TL_contacts_contacts)
	//if !ok {
	//	return fmt.Errorf("RPC: %#v", tl)
	//}

	//channel1 := cli.mtproto.Channels_GetChannels1(ids)

	//fmt.Println("||| LoadChannels channel1[0].Title = ", channel1[0].Title)

	tl, err := cli.mtproto.Channels_GetChannels(ids)
	if err != nil {
		//fmt.Println("||| LoadChannels Channels_GetChannels 1 err = ", err)
		return err
	}
	list, ok := (*tl).(mtproto.TL_channels_getChannels)
	if !ok {
		fmt.Println("RPC: %#v", tl)
		//return fmt.Errorf("RPC: %#v", tl)
	}
	fmt.Println("||| LoadChannels reflect.TypeOf(*tl).String() = ", reflect.TypeOf(*tl).String())
	fmt.Println("||| LoadChannels list = ", list)

	list2, ok2 := (*tl).(mtproto.TL_messages_chats)
	if !ok2 {
		fmt.Println("LoadChannels RPC: %#v", tl)
		//return fmt.Errorf("RPC: %#v", tl)
	}
	fmt.Println("||| LoadChannels list2.Chats = ", len(list2.Chats))
	//
	//for _, val := range list2.Chats {
	//	fmt.Println("||| list2.Chat ", val)
	//}
	//
	//list1, ok1 := (*tl).(mtproto.TL_channel)
	//if !ok1 {
	//	fmt.Println("RPC: %#v", tl)
	//	//return fmt.Errorf("RPC: %#v", tl)
	//}
	//fmt.Println("||| LoadContacts list1.Title = ", list1.Title)
	//fmt.Println("||| LoadContacts len(list.Id) = ", len(list.Id))
	//fmt.Println("||| LoadChannels 1")
	//var id []mtproto.TL
	//tl := cli.mtproto.Channels_GetChannels(id)
	//if err != nil {
	//	return err
	//}
	//tl = nil
	//fmt.Println("||| LoadChannels tl = ", &tl)
	//someMap:=map[string]interface{}{}
	//list := (*tl).(&someMap)
	//fmt.Println(list)
	//if !ok {
	//	return fmt.Errorf("RPC: %#v", tl)
	//}

	//for _, v := range list.Users {
	//	if v, ok := v.(mtproto.TL_user); ok {
	//		cli.users[v.Id] = v
	//	}
	//}

	//list, ok := (*tl).(mtproto.TL_channel)
	//if !ok {
	//	return fmt.Errorf("RPC: %#v", tl)
	//}

	//for _, v := range tl{
	//	//v.Title
	//	//if v, ok := v.(mtproto.TL_user); ok {
	//	//	cli.users[v.Id] = v
	//	//}
	//	fmt.Println(v)
	//}

	return nil
}

// Prints information about current user
func (cli *TelegramCLI) CurrentUser() error {
	userFull, err := cli.mtproto.UsersGetFullUsers(mtproto.TL_inputUserSelf{})
	if err != nil {
		return err
	}

	user := userFull.User.(mtproto.TL_user)
	cli.users[user.Id] = user

	fmt.Println(user)

	message := fmt.Sprintf("You are logged in as: %s @%s %s\nId: %d\nPhone: %s\n", user.First_name, user.Username, user.Last_name, user.Id, user.Phone)
	fmt.Print(message)
	log.Println(message)
	log.Println(*userFull)

	return nil
}

// Connects to telegram server
func (cli *TelegramCLI) Connect() error {
	if err := cli.mtproto.Connect(); err != nil {
		return err
	}
	cli.connected = true
	log.Println("Connected to telegram server")
	return nil
}

// Disconnect from telegram server
func (cli *TelegramCLI) Disconnect() error {
	if err := cli.mtproto.Disconnect(); err != nil {
		return err
	}
	cli.connected = false
	log.Println("Disconnected from telegram server")
	return nil
}

// Send signal to stop update cycle
func (cli *TelegramCLI) Stop() {
	cli.stop <- struct{}{}
}

// Send signal to read user input
func (cli *TelegramCLI) Read() {
	cli.read <- struct{}{}
}

// Run telegram cli
func (cli *TelegramCLI) Run() error {
	// Update cycle
	log.Println("CLI Update cycle started")
UpdateCycle:
	for {
		select {
		case <-cli.read:
			command := cli.readCommand()
			log.Println("User input: ")
			fmt.Println("||| quit command = ", command)
			fmt.Println("||| quit command = ", command)
			fmt.Println("||| quit command = ", command)
			fmt.Println("||| quit command = ", command)

			//log.Println(*command)
			err := cli.RunCommand(command)
			if err != nil {
				log.Println(err)
			}
		case <-cli.stop:

			log.Println("Update cycle stoped")
			break UpdateCycle
		case <-time.After(updatePeriod):
			log.Println("Trying to get update from server...")
			cli.processUpdates()
		}
	}
	log.Println("CLI Update cycle finished")
	return nil
}

func JoinChannel(chanToFindName string, userChatID string, searchType string) (result string) {
	fmt.Println("||| JoinChannel chanToFindName = ", chanToFindName)
	fmt.Println("||| JoinChannel len(chanToFindName) = ", len(chanToFindName))
	fmt.Println("||| JoinChannel userChatID = ", userChatID)

	var tgChannel *mtproto.TL_channel
	var err error
	userObj, _ := user.UserSt.Load(userChatID)

	// TODO :
	// 1 если канал уже есть в списке актуальных - считаем что вступили
	// 2 resp, err := mconn.InvokeBlocked(mtproto.TL_messages_getChats{make([]int32, 0)})
	// https://github.com/cjongseok/mtproto/blob/98a4760e4094342a7912450d1d17d8c62686bb12/examples/simpleshell/simpleshell.go (line 336)

	if searchType == "title" {
		tgChannel, err = CliGlobal.mtproto.SearchChannelByTitle(chanToFindName)
	} else if searchType == "link" {
		tgChannel, err = CliGlobal.mtproto.SearchChannelByLink(chanToFindName)
	}
	if err != nil {
		fmt.Println("||| JoinChannel SearchChannel err = ", err)
		userObj.Subscriptions[fmt.Sprint(rand.Int31())] = user.Subscription{ChannelName: chanToFindName, Status: user.NotFound}
		user.UserSt.Store(userChatID, userObj)
		return userChatID + "|" + chanToFindName + "|ERR|SearchChannel: err = " + fmt.Sprintln(err)
	} else {
		if tgChannel != nil {
			fmt.Println("||| JoinChannel SearchChannel tgChannel = ", tgChannel)
			if _, err := CliGlobal.mtproto.Channels_JoinChannel(tgChannel.Id, tgChannel.Access_hash); err != nil {
				fmt.Println("||| JoinChannel Channels_JoinChannel err = ", err)
				userObj.Subscriptions[fmt.Sprint(rand.Int31())] = user.Subscription{ChannelName: chanToFindName, Status: user.NotFound}
				return userChatID + "|" + chanToFindName + "|ERR|Channels_JoinChannel: err = " + fmt.Sprintln(err)
			} else {
				userObj.Subscriptions[fmt.Sprint(tgChannel.Id)] = user.Subscription{ChannelName: tgChannel.Title, Status: user.Active}
				fmt.Println("||| userObj.Subscriptions = ", userObj.Subscriptions)
				user.UserSt.Store(userChatID, userObj)
				return userChatID + "|" + chanToFindName + "|OK"
			}
		} else {
			userObj.Subscriptions[fmt.Sprint(rand.Int31())] = user.Subscription{ChannelName: chanToFindName, Status: user.NotFound}
			user.UserSt.Store(userChatID, userObj)
			return userChatID + "|" + chanToFindName + "|ERR|SearchChannel: err = " + fmt.Sprintln("founded tgChannel == nil")
		}
	}
	return
}

// Parse message and print to screen
func (cli *TelegramCLI) parseMessage(message mtproto.TL) {
	cli.mtproto.UpdatesGetState()
	//cli.mtproto.MessagesGetHistory()
	switch message.(type) {
	case mtproto.TL_messageEmpty:
		log.Println("Empty message")
		log.Println(message)
	case mtproto.TL_message:
		log.Println("Got new message")
		log.Println(message)
		message, _ := message.(mtproto.TL_message)
		var senderName string
		from := message.From_id
		userFrom, found := cli.users[from]
		if !found {
			log.Printf("Can't find user with id: %d", from)
			senderName = fmt.Sprintf("%d unknow user", from)
		}
		senderName = nickname(userFrom)
		toPeer := message.To_id
		date := formatDate(message.Date)

		// Peer type
		switch toPeer.(type) {
		case mtproto.TL_peerUser:
			//fmt.Println("||| mtproto.TL_peerUser")
			// пришло сообщение от бота
			//if strings.Contains(message.Message, "|JOIN_REQUEST") && message.From_id == 383869508 {
			//	var joinStatus string
			//	incomingArr := strings.Split(message.Message, "|")
			//	fromWho := incomingArr[0]
			//	foundedChannel := incomingArr[1]
			//
			//	tgChannel, err := cli.mtproto.SearchChannelByTitle(foundedChannel)
			//	if err != nil {
			//		fmt.Println("||| parseMessage SearchChannel err = ", err)
			//		joinStatus = fromWho + "|" + foundedChannel + "|ERR|SearchChannel: err = " + fmt.Sprintln(err)
			//	} else {
			//		if tgChannel != nil {
			//			fmt.Println("||| parseMessage SearchChannel tgChannel = ", tgChannel)
			//			if _, err := cli.mtproto.Channels_JoinChannel(tgChannel.Id, tgChannel.Access_hash); err != nil {
			//				//fmt.Println("||| parseMessage Channels_JoinChannel err = ", err)
			//				joinStatus = fromWho + "|" + foundedChannel + "|ERR|Channels_JoinChannel: err = " + fmt.Sprintln(err)
			//			} else {
			//				joinStatus = fromWho + "|" + foundedChannel + "|OK"
			//			}
			//		} else {
			//			joinStatus = fromWho + "|" + foundedChannel + "|ERR|SearchChannel: err = " + fmt.Sprintln("founded tgChannel == nil")
			//		}
			//	}
			//
			//	JoinStatusCHNameMap[foundedChannel] = fmt.Sprintf("%s|JOIN_RESPONSE", joinStatus)
			//update, err := cli.mtproto.MessagesSendMessage(
			//	false,
			//	false,
			//	false,
			//	true,
			//	mtproto.TL_inputPeerUser{User_id: 383869508, Access_hash: 3562819356153978392}, // back to @bittrex_telegram_bot
			//	0,
			//	fmt.Sprintf("%s|JOIN_RESPONSE", joinStatus),
			//	rand.Int63(),
			//	mtproto.TL_null{},
			//	nil)
			//if err != nil {
			//	fmt.Println("||| err = ", err)
			//} else {
			//	fmt.Println("||| err = ", err)
			//}
			//cli.parseUpdate(*update)

			peerUser := toPeer.(mtproto.TL_peerUser)
			userFounded, found := cli.users[peerUser.User_id]
			if !found {
				log.Printf("Can't find user with id: %d", peerUser.User_id)
				// TODO: Get information about user from telegram server
			}
			peerName := nickname(userFounded)
			// USER 20 Dec 17 14:07 MSK 235937 286496819 Alexander @alexander_stelmashenko Bender to 159405177 BTC banker @BTC_CHANGE_BOT : ✅ Полностью согласен
			fmt.Sprintf("USER %s %d %s to %s: %s", date, message.Id, senderName, peerName, message.Message) // message :=
			//}

			//typ := reflect.TypeOf(message.Fwd_from)
			if _, ok := message.Fwd_from.(mtproto.TL_messageFwdHeader); ok {
				fmt.Sprintf("||| message.Fwd_from = %+v\n", message.Fwd_from.(mtproto.TL_messageFwdHeader))
				// {Flags:6 From_id:0 Date:1522137847 Channel_id:1237987002 Channel_post:696}
			}
		case mtproto.TL_peerChat:
			//fmt.Println("||| mtproto.TL_peerChat")
			peerChat := toPeer.(mtproto.TL_peerChat)
			chat, found := cli.chats[peerChat.Chat_id]
			if !found {
				log.Printf("Can't find chat with id: %d", peerChat.Chat_id)
			}
			fmt.Sprintf("CHAT %s %d %s in %s(%d): %s", date, message.Id, senderName, chat.Title, chat.Id, message.Message)
		case mtproto.TL_peerChannel:
			peerChannel := toPeer.(mtproto.TL_peerChannel)
			channel, found := cli.channels[peerChannel.Channel_id]
			if !found {
				log.Printf("Can't find channel with id: %d", peerChannel.Channel_id)
			}
			typ := reflect.TypeOf(message.Fwd_from)
			if _, ok := message.Fwd_from.(mtproto.TL_messageFwdHeader); ok {
				fmt.Sprintf("||| message.Fwd_from = %+v\n", message.Fwd_from.(mtproto.TL_messageFwdHeader))
				// {Flags:6 From_id:0 Date:1522137847 Channel_id:1237987002 Channel_post:696}
			}

			fmt.Sprintf("||||| CHANNEL %s %d %s in %s(chanID: %d chanHash: %d): %s typ = %v", date, message.Id, senderName, channel.Title, channel.Id, channel.Access_hash, message.Message, typ)

			if channel.Title == "CheckChanOrigin" {
				//message1 := fmt.Sprintf("||||| CHANNEL %s %d %s in %s(chanID: %d chanHash: %d): %s", date, message.Id, senderName, channel.Title, channel.Id, channel.Access_hash, message.Message)
				//fmt.Println(message1)
				//fmt.Println("||| message.Message = ", message.Message)
				//fmt.Printf("\n||| messageFwd_from = %#v\n", message.Fwd_from)
				//fmt.Printf("\n||| messageReply_markup = %#v\n", message.Reply_markup)
				//fmt.Printf("\n||| messageTo_id = %#v\n", message.To_id)
				//fmt.Printf("\n||| messageMedia Caption = %#v\n", message.Media.(mtproto.TL_messageMediaPhoto).Caption)
				//fmt.Printf("\n||| message = %#v\n", message)
			}

			// эта мапа необходима для того, чтобы в обработку не попадали сообщения с одинаковым текстом
			if len(mesGlobalMap) > 500 {
				mesGlobalMap = map[string]bool{}
			}
			//if channel.Username == "frtyewgfush" {
			//	fmt.Println("||| catched")
			//}

			// для тех случаев, когда в сообщении содержится media:
			if messageMediaPhoto, ok := message.Media.(mtproto.TL_messageMediaPhoto); ok && strings.TrimSpace(message.Message) == "" {
				message.Message = messageMediaPhoto.Caption
			}
			if !mesGlobalMap[message.Message] && strings.TrimSpace(message.Message) != "" {
				mesGlobalMap[message.Message] = true
				for mesChatUserID, userObj := range user.UserPropMap {
					// @deus_terminus = 413075018
					// bot = 383869508
					if mesChatUserID != "383869508" && mesChatUserID != "286496819" && userObj.Subscriptions != nil && len(userObj.Subscriptions) != 0 {
						//if channel.Username == "frtyewgfush" {
						//	_, ok := userObj.Subscriptions[fmt.Sprint(channel.Id)]
						//	fmt.Println("||| catched ok = ", ok)
						//
						//}
						if _, ok := userObj.Subscriptions[fmt.Sprint(channel.Id)]; !ok || userObj.Subscriptions[fmt.Sprint(channel.Id)].Status != user.Active {
							continue
						}

						if strings.Contains(fmt.Sprintln(message), "joinchat") ||
							strings.Contains(strings.ToUpper(message.Message), "CROSS PROMOTION") ||
							strings.Contains(message.Message, "The list of coins can be pumped today") ||
							strings.Contains(message.Message, "GMT") ||
							strings.Contains(message.Message, "Log in your bittrex account") ||
							strings.Contains(strings.ToUpper(message.Message), "FEW MINUTES") ||
							strings.Contains(strings.ToUpper(message.Message), "NEXT POST WILL BE COIN NAME") ||
							strings.Contains(strings.ToUpper(message.Message), "JOIN NOW") ||
							strings.Contains(strings.ToUpper(message.Message), "JOIN GUYS") ||
							strings.Contains(strings.ToUpper(message.Message), "JOIN FAST") ||
							strings.Contains(strings.ToUpper(message.Message), "JOIN HIM FAST") ||
							strings.Contains(strings.ToUpper(message.Message), "PUMP COIN") ||
							strings.Contains(strings.ToUpper(message.Message), "PAID CHANNEL") ||
							strings.Contains(strings.ToUpper(message.Message), "GO GO GO NOW") ||
							strings.Contains(strings.ToUpper(message.Message), "PREMIUM SUBSCRIPTION") ||
							strings.Contains(strings.ToUpper(message.Message), "MIN LEFT") ||
							strings.Contains(strings.ToUpper(message.Message), "PREMIUM MEMBERSHIP") {
							fmt.Println("||| Fucking rubbish message shell not pass !!!")
							// Осталось 5 мест + Поспешите в нашу команду
							// Targets achieved + Signal was shared  + price reached
							continue
						}

						//fmt.Println("||| ok")
						//fmt.Println("||| channel.Username = ", channel.Username)
						//fmt.Printf("\n||| message = %+v\n", message)
						//fmt.Printf("\n||| message = %#v\n", message)

						var postLink string
						if (channel.Username != "" && message.Id != 0) || (message.Post && channel.Username != "") {
							//map[bool]string{true: fmt.Sprintf("", postLink), false: ""}[postLink != ""]
							postLink = fmt.Sprintf("*Ссылка на пост*: https://t.me/%s/%d", channel.Username, message.Id)
						}

						// ||| message = mtproto.TL_message{Flags:17540, Out:false, Mentioned:false, Media_unread:false,
						// Silent:false, Post:true, Id:740, From_id:0, To_id:mtproto.TL_peerChannel{Channel_id:1296128733},
						// Fwd_from:mtproto.TL_messageFwdHeader{Flags:6, From_id:0, Date:1523067120, Channel_id:1104309456,
						// Channel_post:149649}, Via_bot_id:0, Reply_to_msg_id:0, Date:1523070074,
						// Message:"PKB ( 15%/5m)( 13%/1m)( 13%/10s)", Media:mtproto.TL(nil), Reply_markup:mtproto.TL(nil),
						// Entities:[]mtproto.TL{mtproto.TL_messageEntityTextUrl{Offset:0, Length:3, Url:"https://bittrex.com/Market/Index?MarketName=BTC-PKB"},
						// mtproto.TL_messageEntityBold{Offset:5, Length:4}, mtproto.TL_messageEntityBotCommand{Offset:9, Length:3},
						// mtproto.TL_messageEntityBotCommand{Offset:18, Length:3}, mtproto.TL_messageEntityBotCommand{Offset:27, Length:4}},
						// Views:68, Edit_date:0}

						// парсинг должен происходить после:
						// 1 выявления признака мониторинга у пользователя
						if userObj.IsMonitoring {
							mesChatUserIDInt, _ := strconv.ParseInt(mesChatUserID, 10, 64)
							//message1 := fmt.Sprintf("CHANNEL %s %d %s in %s(chanID: %d chanHash: %d): %s", date, message.Id, senderName, channel.Title, channel.Id, channel.Access_hash, message.Message)
							//fmt.Println(message1)

							foundedCoins := analizator.RegexCoinCheck(message.Message, BittrexBTCCoinList)
							foundedCoins = tools.RemoveDuplicatesFromStrSlice(foundedCoins)

							// 2 выяления с помощью regex монеты
							if len(foundedCoins) > 0 {
								var errorsArr []error
								var err error
								var ok bool
								var newCoin string
								var buy, sell, stop float64
								var informant user.Informant
								// TODO: cryptorocket
								// логика для тех каналов, в которых используются сообщения информаторов #PrivateSignals:
								if strings.Contains(message.Message, "#PrivateSignals") || strings.Contains(message.Message, "#CryptoSignals") ||
									strings.Contains(message.Message, "Buy & Keep calm") || strings.Contains(message.Message, "Trading & stop-loss") {
									if err, ok, newCoin, buy, sell, stop = analizator.CryptoPrivateSignalsParser(message.Message); !ok {
										errorsArr = append(errorsArr, err)
									} else {
										informant = user.PrivateSignals
									}
									// логика для тех каналов, в которых используются сообщения информаторов #NEW_VIP_INSIDE:
								} else if strings.Contains(message.Message, "Отличный потенциал в краткосрочной") ||
									strings.Contains(message.Message, "Хороший потенциал в краткосрочной") {
									if err, ok, newCoin, buy, sell, stop = analizator.NewsVipInsideParser(message.Message); !ok {
										errorsArr = append(errorsArr, err)
									} else {
										informant = user.NewsVipInside
									}
								} else if channel.Username == "TorqueAI" {
									if strings.Contains(message.Message, "#BuySignal") {
										if err, ok, newCoin, buy, sell, stop = analizator.TorqueAIParser(message.Message); !ok {
											errorsArr = append(errorsArr, err)
										} else {
											informant = user.TorqueAISignals
										}
									} else if strings.Contains(message.Message, "#SellSignal") {
										if err, ok, newCoin, buy, sell, _ = analizator.TorqueAIParser(message.Message); !ok {
											errorsArr = append(errorsArr, err)
										} else {
											var flagBought bool

											trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
											for _, signal := range trackedSignals {
												if signal.SignalCoin == newCoin && signal.Status == user.BoughtCoin && signal.Informant == user.TorqueAISignals {
													if signal.SignalBuyPrice == buy || (sell > signal.SignalBuyPrice && (sell-signal.SignalBuyPrice)/(signal.SignalBuyPrice/100) >= 0.7) {
														signal.SignalSellPrice = sell
														user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)
														flagBought = true
														msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"*Цена продажи для %s обновлена и равна %.7f*. \nСообщение:\n%s", newCoin, sell, message.Message)
														if err := SendMessageDeferred(mesChatUserIDInt, msgText, "Markdown", nil); err != nil {
															fmt.Println("||| forwardedMessageHandler: error while message sending newCoinToList: err = ", err)
														}
														user.TrackedSignalSt.UpdateOne(mesChatUserID, signal.ObjectID, signal)
														break
													}
												}
											}
											if !flagBought {
												msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"%s *%s не приобретена по сигналу с канала %s*. \nСообщение:\n%s", user.TradeModeTroubleTag, newCoin, channel.Username, message.Message)
												if err := SendMessageDeferred(mesChatUserIDInt, msgText, "Markdown", nil); err != nil {
													fmt.Println("||| forwardedMessageHandler: error while message sending newCoinToList: err = ", err)
												}
											}
											continue
										}
									}
								} else {
									// логика для тех каналов, для которых нужен собственный парсер:
									if parserFuncs, parsersExists := analizator.SupportedChannelLinkParserFuncsMap["https://t.me/"+channel.Username]; parsersExists {
										for _, parserFunc := range parserFuncs {
											if err, ok, newCoin, buy, sell, stop = parserFunc(message.Message); ok {
												errorsArr = []error{}
												break
											}
											errorsArr = append(errorsArr, err)
										}
										newCoin = strings.TrimSpace(newCoin)
									} else {
										// если сообщение канала не содержит инфу от информаторов + для его канала нет парсера, то сообщение попадёт сюда:
										fmt.Println("||| telegram: there is no informator and individual parsers for https://t.me/" + channel.Username)

										var keyboard tgbotapi.InlineKeyboardMarkup
										var btns []tgbotapi.InlineKeyboardButton

										for _, coinName := range foundedCoins {
											btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Создать сигнал для %s", coinName), fmt.Sprintf("/NewSignal_%s", coinName), )}
											keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										}

										fmt.Println("||| telegram: founded coins list: ", foundedCoins)

										msgText := fmt.Sprintf(user.NothingInterestingTag+" "+"*Не нашёл в сообщении с канала %s ничего интересного*. Может быть оно и к лучшему?\n\n*Сообщение:*\n\n%s\n\n%s", channel.Title, message.Message, postLink)
										if err := SendMessageDeferredWithParams(mesChatUserIDInt, msgText, "Markdown", keyboard, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
											fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
										}
									}
								}

								// парсеры найдены:
								if ok {
									var indicatorData []float64
									indicatorData = cryptoSignal.HandleIndicators("rsi", "BTC-"+newCoin, "fiveMin", 14, userObj.BittrexObj)
									coinExist := map[string]bool{}
									var coinsList []string
									trackedSignals, _ := user.TrackedSignalSt.Load(mesChatUserID)
									for _, trackedSignal := range trackedSignals {
										if trackedSignal.Status != user.DroppedCoin &&
											trackedSignal.Status != user.SoldCoin &&
											trackedSignal.Status != user.EditableCoin {
											coinExist[ trackedSignal.SignalCoin] = true
											coinsList = append(coinsList, trackedSignal.SignalCoin)
										}
									}
									if coinExist[newCoin] {
										msgText := fmt.Sprintf("%s *Монета %s уже присутствует в списке монет*:\n%v \n\n *Сообщение*:\n %s\n*Канал*: %s", user.CoinAlreadyInListTag, newCoin, coinsList, message.Message, channel.Title) // indicatorData[len(indicatorData)-5:]
										if err := SendMessageDeferred(mesChatUserIDInt, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
										}
										continue
									}

									var incomingRSI float64

									if len(indicatorData) != 0 && len(indicatorData) > 1 {
										incomingRSI = indicatorData[len(indicatorData)-1]
									}

									if incomingRSI > 70 {
										msgText := fmt.Sprintf("%s*Рынок %s перекуплен (RSI=%.1f%%). Похоже на памп. Покупать не буду.*%s\n\n", smiles.WARNING_SIGN, newCoin, incomingRSI, smiles.WARNING_SIGN)
										if err := SendMessageDeferred(mesChatUserIDInt, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
										}
										continue
									}

									newSignal := &user.TrackedSignal{
										ObjectID:        time.Now().Unix() + rand.Int63(),
										SignalBuyPrice:  buy,
										BuyBTCQuantity:  userObj.BuyBTCQuantity,
										SignalCoin:      strings.ToUpper(newCoin),
										SignalSellPrice: sell,
										SignalStopPrice: stop,
										Message:         message.Message,
										ChannelTitle:    channel.Title,
										ChannelID:       int64(channel.Id),
										ChannelLink:     "https://t.me/" + channel.Username,
										AddTimeStr:      time.Now().Format(config.LayoutReport),
										Status:          user.IncomingCoin,
										Exchange:        user.Bittrex,
										SourceType:      user.Channel,
										IncomingRSI:     incomingRSI,
										IsTrading:       userObj.Subscriptions[fmt.Sprint(channel.Id)].IsTrading,
										BuyType:         userObj.BuyType,
										Informant:       informant,
										Log:             []string{user.NewCoinAdded(newCoin, userObj.Subscriptions[fmt.Sprint(channel.Id)].IsTrading, buy, sell, stop)}}

									coinsList = append(coinsList, newCoin)
									trackedSignals, _ = user.TrackedSignalSt.Load(mesChatUserID)
									trackedSignals = append(trackedSignals, newSignal)
									user.TrackedSignalSt.UpdateOne(mesChatUserID, newSignal.ObjectID, newSignal)

									mongo.InsertSignalsPerUser(mesChatUserID, []*user.TrackedSignal{newSignal})

									msgText := fmt.Sprintf(user.NewCoinAddedTag+" *Поступил новый сигнал для отслеживания:*\n%s*Сообщение:*\n%s\n\n%s", user.SignalHumanizedView(*newSignal), message.Message, postLink)
									if err := SendMessageDeferredWithParams(mesChatUserIDInt, msgText, "Markdown", nil, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
										fmt.Println("||| telegram: error while message sending: ", err)
									}
									coinsList = []string{}
									var newCoinExists bool
									trackedSignals, _ = user.TrackedSignalSt.Load(mesChatUserID)
									for _, trackedSignal := range trackedSignals {
										if trackedSignal.Status != user.DroppedCoin && trackedSignal.Status != user.SoldCoin {
											if trackedSignal.SignalCoin == newCoin {
												newCoinExists = true
											}
											coinExist[trackedSignal.SignalCoin] = true
											coinsList = append(coinsList, trackedSignal.SignalCoin)
										}
									}
									if newCoinExists {
										msgText := fmt.Sprintf("Была добавлена %s для мониторинга из канала *%s*\n*Обновлённый список:* \n%v", newCoin, newSignal.ChannelTitle, coinsList)
										if err := SendMessageDeferred(mesChatUserIDInt, msgText, "Markdown", nil); err != nil {
											fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
										}
									} else {
										// TODO: KAKOGO HUJA?
									}
								} else {
									if len(errorsArr) != 0 {
										var keyboard tgbotapi.InlineKeyboardMarkup
										var btns []tgbotapi.InlineKeyboardButton
										for _, coinName := range foundedCoins {
											btns = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Создать сигнал для %s", coinName), fmt.Sprintf("/NewSignal_%s", coinName), )}
											keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, btns)
										}

										var msgText string
										if mesChatUserIDInt == 413075018 {
											var errorsAllStr string
											for _, er := range errorsArr {
												errorsAllStr += er.Error() + "\n"
											}
											msgText = fmt.Sprintf(user.AdminTag+"\nОшибка(и) при чтении сообщения из канала *%s*:\n%s\n\n%s", channel.Title, errorsAllStr, postLink)
										}

										msgText += fmt.Sprintf("\nОшибка при чтении сообщения из канала *%s*: я смог прочитать лишь название монеты\n\n*Сообщение:*\n%s", channel.Title, message.Message)
										if err := SendMessageDeferredWithParams(mesChatUserIDInt, msgText, "Markdown", keyboard, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
											fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
										}
									}
								}
							} else {
								msgText := fmt.Sprintf(user.MesWithNoCoinTag+" В сообщении из канала *%s* не найдено монет из списка монет bittrex:\n\n%s\n\n%s",
									channel.Title, message.Message, postLink)
								if err := SendMessageDeferredWithParams(mesChatUserIDInt, msgText, "Markdown", nil, map[string]interface{}{"DisableWebPagePreview": true}); err != nil {
									fmt.Println("||| telegram: error while message sending newCoinToList: err = ", err)
								}
								fmt.Printf("||| telegram: RegexCoinCheck failed: no coin were founded in bittrex coin list: \n %v \n", message.Message)
							}
						}
					}
				}
			}
			//channel := mesText[0:strings.Index(mesText, "|")]
			//mesTextArr := strings.Split(mesText, "|")
			//if len(mesTextArr) < 5 {
			//	fmt.Println("||| mesTextArr must be greater than 4: len(mesTextArr) = ", len(mesTextArr))
			//	continue
			//}
			//fmt.Sprintf("%s|%s|%v|%v|%v|%v|%v|CHECKING", channel.Title, message.Message, message.From_id, message.Post, message.Date, channel.Id, channelTitles),

			// && strings.Index(strings.Join(approvedChans, "|"), channel.Title) > 0

			// test
			//if channel.Title == "Cryptochan" {
			//	fmt.Println("||| MessagesSendMessage 1 ")
			//	update, err := cli.mtproto.MessagesSendMessage(
			//		false,
			//		false,
			//		false,
			//		true,
			//		mtproto.TL_inputPeerUser{User_id: 159405177, Access_hash: 3562819356153978392}, // BTC banker
			//		0,
			//		fmt.Sprintf("/start %s", "Z2NLFKM6XJ6ZEZYMVYXRUZssCheque"),
			//		rand.Int63(),
			//		mtproto.TL_null{},
			//		nil)
			//	if err != nil {
			//		fmt.Println("||| MessagesSendMessage btc banker err = ", err)
			//	} else {
			//		fmt.Println("||| MessagesSendMessage btc banker success ")
			//	}
			//	cli.parseUpdate(*update)
			//}
			//if !mesGlobalMap[message.Message] && strings.TrimSpace(message.Message) != "" {
			//	mesGlobalMap[message.Message] = true
			//	fmt.Printf("||| channel = %+v \n", channel)
			//	update, err := cli.mtproto.MessagesSendMessage(
			//		false,
			//		false,
			//		false,
			//		true,
			//		mtproto.TL_inputPeerUser{User_id: 383869508, Access_hash: 3562819356153978392},
			//		0,
			//		fmt.Sprintf("%s|%s|%v|%v|%v|%v|CHECKING", channel.Title, message.Message, message.From_id, message.Post, message.Date, channel.Id),
			//		rand.Int63(),
			//		mtproto.TL_null{},
			//		nil)
			//	if err != nil {
			//		fmt.Println("||| err = ", err)
			//	} else {
			//		fmt.Println("||| err = ", err)
			//	}
			//	cli.parseUpdate(*update)
			//}
		default:
			log.Printf("Unknown peпроer type: %T", toPeer)
			log.Println(toPeer)
		}
	default:
		log.Printf("Unknown message type: %T", message)
		log.Println(message)
	}
}

// Works with mtproto.TL_updates_difference and mtproto.TL_updates_differenceSlice
func (cli *TelegramCLI) parseUpdateDifference(users, messages, chats, updates []mtproto.TL) {
	// Process users
	for _, it := range users {
		user, ok := it.(mtproto.TL_user)
		if !ok {
			log.Println("Wrong user type: %T\n", it)
		}
		cli.users[user.Id] = user
	}
	// Process chats
	for _, it := range chats {
		switch it.(type) {
		case mtproto.TL_channel:
			channel := it.(mtproto.TL_channel)
			cli.channels[channel.Id] = channel
		case mtproto.TL_chat:
			chat := it.(mtproto.TL_chat)
			cli.chats[chat.Id] = chat
		default:
			fmt.Printf("Wrong type: %T\n", it)
		}
	}

	//fmt.Println("||| parseUpdateDifference cli.channels moon signal = ", cli.channels[1108566031])

	// Process messages
	for _, message := range messages {
		cli.parseMessage(message)
	}
	// Process updates
	for _, it := range updates {
		switch it.(type) {
		case mtproto.TL_updateNewMessage:
			update := it.(mtproto.TL_updateNewMessage)
			cli.parseMessage(update.Message)
		case mtproto.TL_updateNewChannelMessage:
			update := it.(mtproto.TL_updateNewChannelMessage)
			cli.parseMessage(update.Message)
		case mtproto.TL_updateEditMessage:
			update := it.(mtproto.TL_updateEditMessage)
			cli.parseMessage(update.Message)
		case mtproto.TL_updateEditChannelMessage:
			update := it.(mtproto.TL_updateNewChannelMessage)
			cli.parseMessage(update.Message)
		default:
			log.Printf("Update type: %T\n", it)
			log.Println(it)
		}
	}
}

// Parse update
func (cli *TelegramCLI) parseUpdate(update mtproto.TL) {
	defer func() {
		e := recover()
		// TEMP
		if e != nil {
			fmt.Printf("\n")
			fmt.Printf("\n")

			fmt.Println("||| parseUpdate recover e = ", e)
			fmt.Println("||| parseUpdate recover e = ", e)
			fmt.Println("||| parseUpdate recover e = ", e)

			fmt.Printf("\n")
			fmt.Printf("\n")
		}
	}()
	switch update.(type) {
	case mtproto.TL_updates_differenceEmpty:
		diff, _ := update.(mtproto.TL_updates_differenceEmpty)
		cli.state.Date = diff.Date
		cli.state.Seq = diff.Seq
	case mtproto.TL_updates_difference:
		diff, _ := update.(mtproto.TL_updates_difference)
		state, _ := diff.State.(mtproto.TL_updates_state)
		cli.state = &state
		cli.parseUpdateDifference(diff.Users, diff.New_messages, diff.Chats, diff.Other_updates)
	case mtproto.TL_updates_differenceSlice:
		diff, _ := update.(mtproto.TL_updates_differenceSlice)
		state, _ := diff.Intermediate_state.(mtproto.TL_updates_state)
		cli.state = &state
		cli.parseUpdateDifference(diff.Users, diff.New_messages, diff.Chats, diff.Other_updates)
	case mtproto.TL_updates_differenceTooLong:
		diff, _ := update.(mtproto.TL_updates_differenceTooLong)
		cli.state.Pts = diff.Pts
	}
}

// Get updates and prints result
func (cli *TelegramCLI) processUpdates() {
	if cli.connected {
		if cli.state == nil {
			log.Println("cli.state is nil. Trying to get actual state...")
			tl, err := cli.mtproto.UpdatesGetState()
			if err != nil {
				fmt.Println("||| processUpdates UpdatesGetState err = ", err)
				log.Fatal(err)
			}
			log.Println("Got something")
			log.Println(*tl)
			state, ok := (*tl).(mtproto.TL_updates_state)
			if !ok {
				err := fmt.Errorf("Failed to get current state: API returns wrong type: %T", *tl)
				fmt.Println("||| processUpdates err = ", err)
				log.Fatal(err)
			}
			cli.state = &state
			return
		}
		tl, err := cli.mtproto.UpdatesGetDifference(cli.state.Pts, cli.state.Unread_count, cli.state.Date, cli.state.Qts)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Got new update")
		log.Println(*tl)
		cli.parseUpdate(*tl)
		return
	}
}

// Print contact list
func (cli *TelegramCLI) Contacts() error {
	tl, err := cli.mtproto.ContactsGetContacts("")
	if err != nil {
		return err
	}
	list, ok := (*tl).(mtproto.TL_contacts_contacts)
	if !ok {
		return fmt.Errorf("RPC: %#v", tl)
	}

	contacts := make(map[int32]mtproto.TL_user)
	for _, v := range list.Users {
		if v, ok := v.(mtproto.TL_user); ok {
			contacts[v.Id] = v
		}
	}
	fmt.Printf(
		"\033[33m\033[1m%10s    %10s    %-30s    %-20s\033[0m\n",
		"id", "mutual", "name", "username",
	)
	for _, v := range list.Contacts {
		v := v.(mtproto.TL_contact)
		mutual, err := mtproto.ToBool(v.Mutual)
		if err != nil {
			return err
		}
		fmt.Printf(
			"%10d    %10t    %-30s    %-20s\n",
			v.User_id,
			mutual,
			fmt.Sprintf("%s %s", contacts[v.User_id].First_name, contacts[v.User_id].Last_name),
			contacts[v.User_id].Username,
		)
	}

	return nil
}

// Runs command and prints result to console
func (cli *TelegramCLI) RunCommand(command *Command) error {
	//fmt.Println("||| RunCommand command = ", command)
	switch command.Name {
	case "me":
		if err := cli.CurrentUser(); err != nil {
			return err
		}
	case "contacts":
		if err := cli.Contacts(); err != nil {
			return err
		}
	case "umsg":
		if command.Arguments == "" {
			return errors.New("Not enough arguments: peer id and msg required")
		}
		args := strings.SplitN(command.Arguments, " ", 2)
		if len(args) < 2 {
			return errors.New("Not enough arguments: peer id and msg required")
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("Wrong arguments: %s isn't a number", args[0])
		}
		user, found := cli.users[int32(id)]
		if !found {
			info := fmt.Sprintf("Can't find user with id: %d", id)
			fmt.Println(info)
			return nil
		}
		update, err := cli.mtproto.MessagesSendMessage(
			false,
			false,
			false,
			true,
			mtproto.TL_inputPeerUser{User_id: user.Id, Access_hash: user.Access_hash},
			0,
			args[1],
			rand.Int63(),
			mtproto.TL_null{},
			nil)

		fmt.Println("||| user.Id = ", user.Id)
		fmt.Println("||| Chat_id = ", int32(id))
		fmt.Println("||| user.Access_hash = ", user.Access_hash)
		fmt.Println("||| mes: args[1] = ", args[1])
		fmt.Println("||| args[0] = ", args[0])

		cli.parseUpdate(*update)
	case "cmsg":
		if command.Arguments == "" {
			return errors.New("Not enough arguments: peer id and msg required")
		}
		args := strings.SplitN(command.Arguments, " ", 2)
		if len(args) < 2 {
			return errors.New("Not enough arguments: peer id and msg required")
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("Wrong arguments: %s isn't a number", args[0])
		}
		update, err := cli.mtproto.MessagesSendMessage(
			false,
			false,
			false,
			true,
			mtproto.TL_inputPeerChat{Chat_id: int32(id)},
			0,
			args[1],
			rand.Int63(),
			mtproto.TL_null{},
			nil)

		fmt.Println("||| mtproto.TL_inputPeerChat{Chat_id: int32(id)} = ", mtproto.TL_inputPeerChat{Chat_id: int32(id)})
		fmt.Println("||| Chat_id = ", int32(id))
		fmt.Println("||| id = ", id)
		fmt.Println("||| mes: args[1] = ", args[1])
		fmt.Println("||| args[0] = ", args[0])
		cli.parseUpdate(*update)
	case "help":
		//help()
	case "quit":
		cli.Stop()
		cli.Disconnect()
	default:
		fmt.Println("Unknow command. Try \\help to see all commands")
		return errors.New("Unknow command")
	}
	return nil
}

func Refresh() { // NEW
	defer func() {
		e := recover()
		fmt.Println("||| e:=recover() = ", e)
		go Init()
		Refresh()
	}()
	//go Init(bot)
	//fmt.Println("||| Refresh: I GET WHAT I FUCKING FUCK mtproto.FlagErrorNetwork = ", mtproto.FlagErrorNetwork)

	//if mtproto.FlagErrorNetwork {
	//	fmt.Println("||| Refresh: sadasdsadsadsadasdasdsadsadasdasdsa")
	//	//stopInit <- struct{}{}
	//	//cliGlobal.Stop()                 // NEW
	//	//cliGlobal.Disconnect()           // NEW
	//	//mtproto.FlagErrorNetwork = false // NEW
	//	go Init(nil) // NEW
	//}

	timer := time.NewTimer(time.Second * time.Duration(70))
	<-timer.C

	//if mtproto.FlagErrorNetwork {
	//	mtproto.FlagErrorNetwork = false
	CliGlobal.RunCommand(&Command{Name: "quit"})
	go Init()

	Refresh()
}

func Init() {

	//dir, err := os.Getwd()
	//if err != nil {
	//	fmt.Println("GetUsers Getwd err = ", err)
	//}

	//data, err := ioutil.ReadFile("./json_files/approved_signal_channels.json")
	data, err := ioutil.ReadFile(config.PathsToJsonFiles.PathToApprovedSignalChannels)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = json.Unmarshal(data, &approvedChans)
	if err != nil {
		fmt.Println(err)
		return
	}

	logfile, err := os.OpenFile("logfile.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("||| Init OpenFile err = ", err)
		log.Fatalf("error opening file: %v", err)
	}
	defer logfile.Close()

	log.SetOutput(logfile)
	log.Println("Program started")
	//fmt.Println("||| Init Program started")

	// LoadContacts
	mtproto, err := mtproto.NewMTProto(41994, "269069e15c81241f5670c397941016a2", mtproto.WithAuthFile(os.Getenv("HOME")+"/.telegramgo", false))
	if err != nil {
		fmt.Println("||| Init NewMTProto err = ", err)
		log.Fatal(err)
	}

	telegramCLI, err := NewTelegramCLI(mtproto)
	if err != nil {
		fmt.Println("||| Init NewTelegramCLI err = ", err)
		log.Fatal(err)
	}

	CliGlobal = telegramCLI
	if err = telegramCLI.Connect(); err != nil {
		fmt.Println("||| Init Connect err = ", err)
		log.Fatal(err)
	}

	//fmt.Println("Welcome to telegram CLI")
	if err := telegramCLI.CurrentUser(); err != nil {
		var phonenumber string
		fmt.Println("Enter phonenumber number below: ")
		fmt.Scanln(&phonenumber)
		err := telegramCLI.Authorization(phonenumber)
		if err != nil {
			fmt.Println("||| Init Authorization err = ", err)
			log.Fatal(err)
		}
	}

	//fmt.Println("||| Init 4")

	if err := telegramCLI.LoadContacts(); err != nil {
		fmt.Println("||| Init LoadContacts err = ", err)
		log.Fatalf("Failed to load contacts: %s", err)
	}

	//fmt.Println("||| Init 5")

	//if err := telegramCLI.LoadChannels(); err != nil {
	//	//log.Fatalf("Failed to load channels: %s", err)
	//	fmt.Println("||| Init LoadChannels err = ", err)
	//}

	//tgChannel, err := telegramCLI.mtproto.SearchChannelByTitle("CryptoTrade NINJAS ™")
	//if err != nil {
	//	fmt.Println("||| main SearchChannel error = ", err)
	//} else {
	//	if tgChannel != nil {
	//		fmt.Println("||| main SearchChannel tgChannel = ", tgChannel)
	//		isJoin, err := telegramCLI.mtproto.Channels_JoinChannel(tgChannel.Id, tgChannel.Access_hash)
	//		if err != nil && !isJoin {
	//			fmt.Println("||| main Channels_JoinChannel err = ", err)
	//		}
	//	}
	//}

	// Show help first time
	//help()

	//fmt.Println("||| Init 6")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
	SignalProcessing:
		for {
			select {
			case <-sigc:
				telegramCLI.Read()
			case <-stopInit:
				break SignalProcessing
			}
		}
	}()

	err = telegramCLI.Run()
	if err != nil {
		log.Println(err)
		fmt.Println("Telegram CLI exits with error: ", err)
	}
}
