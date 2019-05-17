package server

import (
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"math/rand"
	"net/http"
	"proxy/define"
	"proxy/job"
	"proxy/utils"
	"strconv"
	"strings"
	"time"
)

func makeValidateName(name string) string {

	buf := []byte(name)
	var newName strings.Builder
	for i, val := range buf {
		if !isValidNameChar(val) {
			continue
		} else {
			newName.WriteString(string(name[i]))
		}
	}
	return newName.String()
}

func isValidNameChar(c byte) bool {
	if c >= '0' && c <= '9' {
		return true
	} else if c >= 'a' && c <= 'z' {
		return true
	} else if c >= 'A' && c <= 'Z' {
		return true
	} else {
		return false
	}
}

func checkAccountExist(id string) (bool, error) {
	exist, errKey := dbInstance.EXISTS(id)
	return exist, errKey
}

func getAccountName(id string) (string, error) {
	name, errKey := dbInstance.HGETString(id, define.Name)
	return name, errKey
}

func checkType(t int64) bool {
	if t != PhotoGrid && t != Contentos && t != Game2048 {
		return false
	}
	return true
}

/**
 * 2048游戏上链
 */
func game2048(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	cosStr := r.FormValue("cos")
	typeStr := r.FormValue("type")
	gameIdStr := r.FormValue("gameId")
	loserIdStr := r.FormValue("loserId")
	loserCosStr := r.FormValue("loserCos")
	loserNameStr := r.FormValue("loserName")
	winnerIdStr := r.FormValue("winnerId")
	winnerCosStr := r.FormValue("winnerCos")
	winnerNameStr := r.FormValue("winnerName")

	if winnerIdStr == "" ||
		winnerCosStr == "" ||
		winnerNameStr == "" ||
		loserIdStr == "" ||
		loserCosStr == "" ||
		loserNameStr == "" ||
		gameIdStr == "" ||
		typeStr == "" {
		res["ret"] = ParamError
		return
	}

	// check typeid
	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	// params
	cos, cerr := strconv.ParseUint(cosStr, 10, 64)
	winnerCos, wcerr := strconv.ParseUint(winnerCosStr, 10, 64)
	loserCos, lcerr := strconv.ParseUint(loserCosStr, 10, 64)
	// gameId, gerr := strconv.ParseUint(gameIdStr, 10, 64)
	loserId, lerr := strconv.ParseUint(loserIdStr, 10, 64)

	if wcerr != nil || lcerr != nil || lerr != nil || cerr != nil {
		res["ret"] = ParamError
		return
	}

	// check gameid
	gamePrefix := getSpecificPrefix("game", typeInt)
	gameIdStr = gamePrefix + gameIdStr + loserIdStr
	exist, err := dbInstance.EXISTS(gameIdStr)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if exist {
		res["ret"] = GameIdExist
		return
	}

	// if account isnot exist create it
	idPrefix := getSpecificPrefix("id", typeInt)
	namePrefix := getSpecificPrefix("name", typeInt)

	winnerIdStr = idPrefix + winnerIdStr
	winnerNameStr = namePrefix + winnerNameStr
	newWinnerNameStr := makeValidateName(winnerNameStr)
	if len(newWinnerNameStr) != 0 {
		winnerNameStr = utils.GenerateName(newWinnerNameStr)
	} else {
		len := rand.Intn(8) + 8
		winnerNameStr = utils.RandStringBytes(len)
	}

	loserIdStr = idPrefix + loserIdStr
	loserNameStr = namePrefix + loserNameStr
	newLoserNameStr := makeValidateName(loserNameStr)
	if len(newLoserNameStr) != 0 {
		loserNameStr = utils.GenerateName(newLoserNameStr)
	} else {
		len := rand.Intn(8) + 8
		loserNameStr = utils.RandStringBytes(len)
	}

	msg := &job.Game2048Msg{
		Wid:   winnerIdStr,
		Wname: winnerNameStr,
		Wcos:  winnerCos,
		Lid:   loserIdStr,
		Lname: loserNameStr,
		Lcos:  loserCos,
		Cos:   cos,
		Gid:   gameIdStr}
	msg.AppStr = getSpecificPrefix("", typeInt)
	sendMsg(loserId, msg)

	res["ret"] = OK
	return
}

/**
 * 用户链上行为列表
 */
func actionList(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")
	typeStr := r.FormValue("type")
	if id == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	// check id
	// exist, err := checkAccountExist(id)
	name, err := getAccountName(id)
	//name = "initminer"
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if name == "" {
		res["ret"] = IdNotExist
		return
	}

	// result type
	result := make([]struct {
		TimeStamp string
		Action    string
		TxHash    string
	}, 5)

	// get list by contentos-go/GetUserTrxListByTime
	path := userId % uint64(jobCount)
	list, _ := jobs[path].GetUserActionList(name)
	if list != nil {
		data := list.GetTrxList()
		// get item
		for i, v := range data {
			if op := v.GetTrxWrap().GetSigTrx().GetTrx().GetOperations(); op != nil {
				result[i].Action = getActionName(op[0])
			}
			result[i].TxHash = v.GetTrxId().ToString()
			result[i].TimeStamp = v.GetBlockTime().ToString()
		}
	}

	res["list"] = result
	res["ret"] = OK
	return
}

func getActionName(op *prototype.Operation) string {

	if a := op.GetOp6(); a != nil {
		return "Post" // 上传视频
	} else if a := op.GetOp7(); a != nil {
		return "Comment" // 评论
	} else if a := op.GetOp8(); a != nil {
		return "Follow" // 关注
	} else if a := op.GetOp9(); a != nil {
		return "Like" // 点赞
	} else if a := op.GetOp14(); a != nil {
		switch a.GetMethod() {
		case "checkincount": // 签到
			return "Signin"
		default:
			return "Signin"
		}
	}

	return "otherAction"
}

/**
 * 签到
 */
func signIn(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")        // 用户uid
	dateStr := r.FormValue("date") // 签到的时间戳
	typeStr := r.FormValue("type") // 项目类型: contento [ type = 2 ]

	if id == "" || dateStr == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	datePrefix := getSpecificPrefix("date", typeInt)
	dateStr = datePrefix + dateStr

	// check id
	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}

	isSigned, err := dbInstance.EXISTS(id + dateStr)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if isSigned {
		res["ret"] = Signed
		return
	}

	msg := &job.SignInMsg{Id: id, Date: dateStr}
	msg.AppStr = getSpecificPrefix("", typeInt)

	res["ret"] = OK

	sendMsg(userId, msg)
	return
}

func createAccount(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")
	name := r.FormValue("user_name")
	typeStr := r.FormValue("type")

	if id == "" || name == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	namePrefix := getSpecificPrefix("name", typeInt)
	name = namePrefix + name

	newName := makeValidateName(name)
	if len(newName) != 0 { // got a valid subname
		name = utils.GenerateName(newName)
	} else { // all char invalid
		len := rand.Intn(8) + 8 // 8 ~ 15
		name = utils.RandStringBytes(len)
	}

	// check name exist
	nameExist, errName := dbInstance.EXISTS(name)
	if errName != nil {
		res["ret"] = ServerError
		return
	}
	if nameExist {
		res["ret"] = ServerError
		return
	}

	// check exist
	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if exist {
		res["ret"] = IdDuplicate
		return
	}

	res["ret"] = OK

	msg := &job.AccountMsg{Id: id, Name: name}
	msg.AppStr = getSpecificPrefix("", typeInt)

	sendMsg(userId, msg)
	return
}

func post(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")
	postIdStr := r.FormValue("post_id")
	title := r.FormValue("title")
	content := r.FormValue("content")
	tag := r.FormValue("tag")
	typeStr := r.FormValue("type")

	if postIdStr == "" || id == "" || content == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	// parse postIdStr to uint64
	_, err := strconv.ParseUint(postIdStr, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}
	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	if title == "" {
		defaultTitle := getSpecificPrefix("", typeInt)
		title = defaultTitle
	}

	if tag == "" {
		defaultTag := getSpecificPrefix("", typeInt)
		tag = defaultTag
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	postPrefix := getSpecificPrefix("post", typeInt)
	postIdStr = postPrefix + postIdStr

	msg := &job.PostMsg{Id: id, PostId: postIdStr, Title: title, Content: content, Tag: tag}
	msg.AppStr = getSpecificPrefix("", typeInt)

	// check exist
	postExist, errPost := dbInstance.EXISTS(postIdStr)
	if errPost != nil {
		res["ret"] = ServerError
		return
	}
	if postExist {
		res["ret"] = PostIdDuplicate
		return
	}

	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}

	res["ret"] = OK

	sendMsg(userId, msg)
	return
}

func like(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")
	postIdStr := r.FormValue("post_id")
	typeStr := r.FormValue("type")

	if id == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	postPrefix := getSpecificPrefix("post", typeInt)
	postIdStr = postPrefix + postIdStr

	// check id
	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}
	app := getSpecificPrefix("", typeInt)

	// check post, if post not valid, we send to a fake collector
	postExist, errPost := dbInstance.EXISTS(postIdStr)
	if errPost != nil {
		res["ret"] = ServerError
		return
	}
	if !postExist {
		res["ret"] = PostIdNotExist
		sendFakeLikeMsg(userId, id, app)
		return
	}

	uniqueLike := id + postIdStr
	uniqueLike = getSpecificPrefix("like", typeInt) + uniqueLike
	uniqueLikeExist, err := dbInstance.EXISTS(uniqueLike)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if uniqueLikeExist {
		res["ret"] = LikePostDuplicate
		return
	}

	msg := &job.LikeMsg{Id: id, PostId: postIdStr, UniqueName: uniqueLike}
	msg.AppStr = app

	res["ret"] = OK

	sendMsg(userId, msg)
	return
}

func sendFakeLikeMsg(id uint64, combineId, app string) {
	name, err := dbInstance.HGETString(combineId, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, id, name))
		return
	}
	msg := &job.FakeLikeMsg{Id: combineId, Name: name}
	msg.AppStr = app
	path := id % uint64(jobCount)
	jobs[path].Put(msg)
}

func comment(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()
	id := r.FormValue("id")
	postIdStr := r.FormValue("post_id")
	commentIdStr := r.FormValue("comment_id")
	commentContent := r.FormValue("comment_content")
	typeStr := r.FormValue("type")

	if id == "" || commentIdStr == "" || commentContent == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	// check comment exist
	_, err := strconv.ParseUint(commentIdStr, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	userId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	idPrefix := getSpecificPrefix("id", typeInt)
	id = idPrefix + id

	postPrefix := getSpecificPrefix("post", typeInt)
	postIdStr = postPrefix + postIdStr

	commentPrefix := getSpecificPrefix("comment", typeInt)
	commentIdStr = commentPrefix + commentIdStr

	commentExist, errComment := dbInstance.EXISTS(commentIdStr)
	if errComment != nil {
		res["ret"] = ServerError
		return
	}
	if commentExist {
		res["ret"] = CommentIdDuplicate
		return
	}

	// check name
	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}

	app := getSpecificPrefix("", typeInt)

	// check post id, if post id not exist, we send to a fake collector
	postExist, errPost := dbInstance.EXISTS(postIdStr)
	if errPost != nil {
		res["ret"] = ServerError
		return
	}
	if !postExist {
		res["ret"] = PostIdNotExist
		sendFakeCommentMsg(userId, id, commentContent, app)
		return
	}

	res["ret"] = OK

	msg := &job.CommentMsg{Id: id, PostId: postIdStr, CommentId: commentIdStr, Content: commentContent}
	msg.AppStr = app

	sendMsg(userId, msg)
	return
}

func sendFakeCommentMsg(id uint64, combineId, content, app string) {
	name, err := dbInstance.HGETString(combineId, define.Name)
	if err != nil || name == "" {
		log.Error(fmt.Sprintf("get account name error:%v account:%v name:%v", err, id, name))
		return
	}
	msg := &job.FakeCommentMsg{Id: combineId, Name: name, Content: content}
	msg.AppStr = app
	path := id % uint64(jobCount)
	jobs[path].Put(msg)
}

func follow(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	res["ret"] = OK
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()

	userId, uid, fuid, typeInt := checkFollow(res, r)
	if res["ret"] != OK {
		return
	}

	uniqueFollow := uid + fuid
	uniqueFollow = getSpecificPrefix("follow", typeInt) + uniqueFollow
	followExist, err := dbInstance.EXISTS(uniqueFollow)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if followExist {
		res["ret"] = FollowDuplicate
		return
	}

	msg := &job.FollowMsg{Uid: uid, Fuid: fuid, UniqueFollow: uniqueFollow, Cancel: false}
	msg.AppStr = getSpecificPrefix("", typeInt)

	sendMsg(userId, msg)
	return
}

func unfollow(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pStr := ""
	res := map[string]interface{}{}
	res["ret"] = OK
	defer retPostWriter(r, wr, &pStr, time.Now(), res)
	if err := r.ParseForm(); err != nil {
		log.Error("r.ParseForm() failed(%v)", err)
		res["ret"] = ParamError
		return
	}
	pStr = r.Form.Encode()

	userId, uid, fuid, typeInt := checkFollow(res, r)
	if res["ret"] != OK {
		return
	}

	uniqueFollow := uid + fuid
	uniqueFollow = getSpecificPrefix("follow", typeInt) + uniqueFollow
	followExist, err := dbInstance.EXISTS(uniqueFollow)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !followExist {
		res["ret"] = OK
		return
	}

	msg := &job.FollowMsg{Uid: uid, Fuid: fuid, UniqueFollow: uniqueFollow, Cancel: true}
	msg.AppStr = getSpecificPrefix("", typeInt)

	sendMsg(userId, msg)
	return
}

func checkFollow(res map[string]interface{}, r *http.Request) (uidInt uint64, uid string, fuid string, typeInt int64) {
	uid = r.FormValue("uid")
	fuid = r.FormValue("fuid")
	typeStr := r.FormValue("type")
	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	if uid == "" || fuid == "" || typeStr == "" {
		res["ret"] = ParamError
		return
	}

	uidInt, err = strconv.ParseUint(uid, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	_, err = strconv.ParseUint(fuid, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	uidPrefix := getSpecificPrefix("id", typeInt)
	uid = uidPrefix + uid

	fuidPrefix := getSpecificPrefix("id", typeInt)
	fuid = fuidPrefix + fuid

	if uid == fuid {
		res["ret"] = FollowSelf
		return
	}

	// check uid
	exist, err := checkAccountExist(uid)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}

	// check fuid
	fexist, err := checkAccountExist(fuid)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !fexist {
		res["ret"] = FuidNotExist
		return
	}
	return
}

/**
 * 获取链上注册的用户名
 */
func getName(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(wr, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	res := map[string]interface{}{}
	defer retGetWriter(r, wr, time.Now(), res)
	params := r.URL.Query()

	id := params.Get("id")
	if id == "" {
		res["ret"] = ParamError
		return
	}

	typeStr := r.FormValue("type")
	typeInt, err := strconv.ParseInt(typeStr, 10, 32)
	if err != nil || !checkType(typeInt) {
		res["ret"] = ParamError
		return
	}

	_, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		res["ret"] = ParamError
		return
	}

	uidPrefix := getSpecificPrefix("id", typeInt)
	id = uidPrefix + id

	// check id
	exist, err := checkAccountExist(id)
	if err != nil {
		res["ret"] = ServerError
		return
	}
	if !exist {
		res["ret"] = IdNotExist
		return
	}

	name, err := dbInstance.HGETString(id, define.Name)
	if err != nil || name == "" {
		res["ret"] = ServerError
		return
	}

	res["ret"] = OK
	res["name"] = name
	return
}

/**
 * 将整合数据传递给job处理
 */
func sendMsg(id uint64, m interface{}) {
	path := id % uint64(jobCount)
	jobs[path].Put(m)
}

/**
 * 前缀
 *	params:
 *		api  string   参数字段前缀
 *		t    int64    项目前缀
 *  return:
 *		string
 */
func getSpecificPrefix(api string, t int64) string {
	prefix := ""
	switch api {
	case "id":
		prefix = define.IdPrefix
	case "post":
		prefix = define.PostPrefix
	case "like":
		prefix = define.LikePrefix
	case "comment":
		prefix = define.CommentPrefix
	case "follow":
		prefix = define.FollowPrefix
	case "date":
		prefix = define.DatePrefix
	case "name":
		prefix = define.NamePrefix
	case "game":
		prefix = define.GamePrefix
	default:
	}

	switch t {
	case PhotoGrid:
		prefix += define.PGStr
	case Contentos:
		prefix += define.ContentosStr
	case Game2048:
		prefix += define.Game2048Str
	default:
	}
	return prefix
}
