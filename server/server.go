package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pojol/gobot/bot"
	"github.com/pojol/gobot/bot/behavior"
	"github.com/pojol/gobot/database"
	"github.com/pojol/gobot/factory"
	"github.com/pojol/gobot/utils"
)

type Err int32

const (
	Succ           Err = 200
	Fail           Err = 1000
	ErrContentRead     = 1000 + iota
	ErrWrongInput
	ErrJsonUnmarshal
	ErrJsonInvalid
	ErrPluginLoad
	ErrEnd
	ErrBreak
	ErrCantFindBot
	ErrCreateBot
	ErrEmptyBatch
	ErrTagsFormat
	ErrRunningErr
	ErrUploadConfig
	ErrGetConfig
)

var errmap map[Err]string = map[Err]string{
	ErrContentRead:   "failed to read request content",
	ErrJsonInvalid:   "wrong file format",
	ErrJsonUnmarshal: "json unmarshal err",
	ErrWrongInput:    "bad request parameter",
	ErrPluginLoad:    "failed to plugin load",
	ErrEnd:           "run to the end",
	ErrBreak:         "run to the break",
	ErrCantFindBot:   "can't find bot",
	ErrCreateBot:     "failed to create bot, the behavior tree file needs to be uploaded to the server before creation",
	ErrEmptyBatch:    "empty batch info",
}

func FileBlobUpload(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	code := Succ

	name := ctx.Request().Header.Get("FileName")
	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	if len(bts) == 0 {
		code = ErrContentRead // tmp
		fmt.Println("bytes is empty!")
		goto EXT
	}

	_, err = behavior.Load(bts, behavior.Step)
	if err != nil {
		fmt.Println(err.Error())
		code = ErrJsonInvalid
		goto EXT
	}

	database.GetBehavior().Upset(name, bts)

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]

	ctx.JSON(http.StatusOK, res)
	return nil
}

func FileTextUpload(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	code := Succ
	var upload *utils.UploadFile
	var fbyte []byte
	var name string

	f, header, err := ctx.Request().FormFile("file")
	if err != nil {
		code = ErrContentRead
		fmt.Println(err.Error())
		goto EXT
	}

	upload = utils.NewUploadFile(f, header)
	fbyte = upload.ReadBytes()
	if len(fbyte) == 0 {
		code = ErrContentRead
		goto EXT
	}

	name = upload.FileName()
	_, err = behavior.Load(fbyte, behavior.Step)
	if err != nil {
		fmt.Println(err.Error())
		code = ErrJsonInvalid
		goto EXT
	}

	database.GetBehavior().Upset(name, fbyte)

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]

	ctx.JSON(http.StatusOK, res)
	return nil
}

func PrefabUpload(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}

	name := ctx.Request().Header.Get("FileName")
	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		res.Code = ErrContentRead // tmp
		goto ext
	}

	if name != "" {
		database.GetPrefab().Upset(name, bts)
	}

ext:
	ctx.JSON(http.StatusOK, res)
	return nil
}

func PrefabList(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	body := PrefabListRes{}

	tabs, _ := database.GetPrefab().List()
	for _, v := range tabs {

		tags := []string{}
		json.Unmarshal(v.Tags, &tags)

		body.Lst = append(body.Lst, PrefabInfo{
			Name: v.Name,
			Tags: tags,
		})

	}

	res.Body = body
	ctx.JSON(http.StatusOK, res)
	return nil
}

func PrefabGetInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")

	name := ctx.Request().Header.Get("FileName")

	tab, _ := database.GetPrefab().Find(name)

	ctx.Blob(http.StatusOK, "text/plain;charset=utf-8", tab.Code)
	return nil
}

func PrefabRmv(ctx echo.Context) error {

	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	req := &PrefabRmvReq{}

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		res.Code = ErrContentRead
		goto ext
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		res.Code = ErrJsonInvalid
		fmt.Println(err.Error())
		goto ext
	}

	if req.Name != "" {
		err = database.GetPrefab().Rmv(req.Name)
		if err != nil {
			res.Code = int(Fail)
			res.Msg = err.Error()
		}
	}

ext:
	ctx.JSON(http.StatusOK, res)
	return nil
}

func PrefabSetTags(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	req := &PrefabSetTagsReq{}
	var tagdat []byte

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		res.Code = ErrContentRead
		goto ext
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		res.Code = ErrJsonInvalid
		fmt.Println(err.Error())
		goto ext
	}

	tagdat, err = json.Marshal(req.Tags)
	if err != nil {
		res.Code = ErrTagsFormat
		res.Msg = err.Error()
		goto ext
	}

	fmt.Println("set tags", req.Name, string(tagdat))
	database.GetPrefab().UpdateTags(req.Name, tagdat)

ext:
	ctx.JSON(http.StatusOK, res)
	return nil
}

func FileRemove(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	code := Succ
	res := &Response{}
	req := &FileRemoveReq{}
	body := &FileRemoveRes{}
	var info []database.BehaviorTable

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, req)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	database.GetBehavior().Rmv(req.Name)
	info, err = database.GetBehavior().List()

	for _, v := range info {
		tags := []string{}
		json.Unmarshal(v.Tags, &tags)
		body.Bots = append(body.Bots, behaviorInfo{
			Name:   v.Name,
			Update: v.UpdateTime,
			Status: v.Status,
			Tags:   tags,
		})
	}

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func FileGetList(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	body := &BehaviorListRes{}

	info, err := database.GetBehavior().List()
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
		goto ext
	}

	for _, v := range info {
		tags := []string{}
		json.Unmarshal(v.Tags, &tags)
		body.Bots = append(body.Bots, behaviorInfo{
			Name:   v.Name,
			Update: v.UpdateTime,
			Status: v.Status,
			Tags:   tags,
		})
	}

ext:
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func FileSetTags(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	code := Succ
	res := &Response{}
	req := &SetBehaviorTagsReq{}
	body := &SetBehaviorTagsRes{}
	var info []database.BehaviorTable
	var jdat []byte

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		code = ErrJsonInvalid
		fmt.Println(err.Error())
		goto EXT
	}

	jdat, err = json.Marshal(req.NewTags)
	if err != nil {
		code = ErrTagsFormat
		goto EXT
	}

	database.GetBehavior().UpdateTags(req.Name, jdat)
	info, err = database.GetBehavior().List()
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
		goto EXT
	}

	for _, v := range info {
		tags := []string{}
		json.Unmarshal(v.Tags, &tags)
		body.Bots = append(body.Bots, behaviorInfo{
			Name:   v.Name,
			Update: v.UpdateTime,
			Status: v.Status,
			Tags:   tags,
		})
	}

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func ConfigGetSysInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	body := ConfigGetSysInfoRes{}

	conf, err := database.GetConfig().Get()
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
		goto ext
	}

	body.ChannelSize = conf.ChannelSize
	body.ReportSize = conf.ReportSize
	body.EnqueneDelay = conf.EnqueneDelay

ext:
	res.Body = body
	ctx.JSON(http.StatusOK, res)
	return nil
}

func ConfigSetSysInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	body := ConfigSetSysInfoRes{}
	req := &ConfigSetSysInfoReq{}
	conf := database.GetConfig()
	var newtab database.ConfTable

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

	if req.ChannelSize != 0 {
		conf.UpdateChannelSize(req.ChannelSize)
	}
	if req.ReportSize != 0 {
		conf.UpdateReportSize(req.ReportSize)
	}
	if req.EnqueneDelay != 0 {
		conf.UpdateEnqueneDelay(req.EnqueneDelay)
	}

	newtab, err = conf.Get()
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
		goto EXT
	}

	body.ReportSize = newtab.ReportSize
	body.ChannelSize = newtab.ChannelSize
	body.EnqueneDelay = newtab.EnqueneDelay

EXT:
	res.Body = body
	ctx.JSON(http.StatusOK, res)
	return nil
}

func ConfigGetGlobalInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	code := Succ
	res := &Response{}

	res.Code = int(code)

	conf, _ := database.GetConfig().Get()
	ctx.Blob(http.StatusOK, "text/plain;charset=utf-8", conf.GlobalCode)
	return nil
}

func ConfigSetGlobalInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		res.Code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	err = database.GetConfig().UpdateGlobalDefine(bts)
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
		goto EXT
	}

EXT:
	ctx.JSON(http.StatusOK, res)
	return nil
}

func FileGetBlob(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	req := &FindBehaviorReq{}
	info := database.BehaviorTable{}

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

	info, err = database.GetBehavior().Find(req.Name)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

EXT:
	ctx.Blob(http.StatusOK, "text/plain;charset=utf-8", info.File)
	return nil
}

func GetReport(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{
		Code: int(Succ),
	}
	body := &ReportRes{}
	var err error

	body.Info, err = database.GetReport().List()
	if err != nil {
		res.Code = int(Fail)
		res.Msg = err.Error()
	}

	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func BotRun(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	req := &BotRunRequest{}
	code := Succ
	var info database.BehaviorTable
	var tree *behavior.Tree
	var b *bot.Bot

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		code = ErrJsonUnmarshal
		fmt.Println(err.Error())
		goto EXT
	}

	if req.Name == "" {
		code = ErrWrongInput
		goto EXT
	}
	fmt.Println(req.Name, "bot run block begin")

	info, err = database.GetBehavior().Find(req.Name)
	if err != nil {
		code = Fail
		goto EXT
	}

	tree, err = behavior.Load(info.File, behavior.Block)
	if err != nil {
		code = Fail
		goto EXT
	}
	b = bot.NewWithBehaviorTree("script/", tree, req.Name, "", 1, string(info.File))
	err = b.RunByBlock()
	if err != nil {
		code = ErrRunningErr
		errmap[code] = err.Error()
	}
EXT:
	fmt.Println(req.Name, "bot run block end", err)
	res.Code = int(code)
	res.Msg = errmap[code]
	ctx.JSON(http.StatusOK, res)
	return nil
}

func BotCreateBatch(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	req := &BotBatchCreateRequest{}
	code := Succ

	var err error

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	if req.Name == "" {
		goto EXT
	}
	if req.Num == 0 {
		goto EXT
	}

	err = factory.Global.AddBatch(req.Name, 0, int32(req.Num))
	if err != nil {
		res.Msg = err.Error()
	}

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]

	ctx.JSON(http.StatusOK, res)
	return nil
}

func BotList(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	body := &BotListResponse{}
	code := Succ

	body.Lst = factory.Global.GetBatchInfo()
	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func DebugStep(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	req := &StepRequest{}
	body := &StepResponse{}
	code := Succ
	var b *bot.Bot

	var err error
	var s bot.State

	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	err = json.Unmarshal(bts, &req)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	b = factory.Global.FindBot(req.BotID)
	if b == nil {
		code = ErrCantFindBot
		goto EXT
	}

	s = b.RunByStep()
	body.Blackboard = b.GetMetaInfo()
	body.ThreadInfo = b.GetThreadInfo()

	if s == bot.SEnd {
		code = ErrEnd
		defer factory.Global.RmvBot(req.BotID)
		goto EXT
	} else if s == bot.SBreak {
		code = ErrBreak
		defer factory.Global.RmvBot(req.BotID)
		goto EXT
	}

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func DebugCreate(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	body := &CreateDebugBotResponse{}
	code := Succ
	var b *bot.Bot

	name := ctx.Request().Header.Get("FileName")
	bts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		code = ErrContentRead // tmp
		fmt.Println(err.Error())
		goto EXT
	}

	b = factory.Global.CreateDebugBot(name, bts)
	if b == nil {
		fmt.Println("create bot err", name)
		code = ErrCreateBot
		goto EXT
	}

	body.BotID = b.ID()
	body.ThreadInfo = b.GetThreadInfo()

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

func ReqPrint() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Println(c.Path(), c.Request().Method, "enter")
			err := next(c)
			fmt.Println(c.Path(), c.Request().Method, err)
			return err
		}
	}
}

func Route(e *echo.Echo) {

	e.GET("/health", func(ctx echo.Context) error {
		ctx.JSONBlob(http.StatusOK, []byte(``))
		return nil
	})

	e.POST("/file.uploadTxt", FileTextUpload)
	e.POST("/file.uploadBlob", FileBlobUpload)
	e.POST("/file.remove", FileRemove)
	e.POST("/file.list", FileGetList)
	e.POST("/file.get", FileGetBlob)
	e.POST("/file.setTags", FileSetTags)

	e.POST("/prefab.list", PrefabList)
	e.POST("/prefab.get", PrefabGetInfo)
	e.POST("/prefab.rmv", PrefabRmv)
	e.POST("/prefab.setTags", PrefabSetTags)
	e.POST("/prefab.upload", PrefabUpload)

	e.POST("/bot.run", BotRun)
	e.POST("/bot.batch", BotCreateBatch) // 创建一批bot
	e.POST("/bot.list", BotList)

	e.POST("/debug.create", DebugCreate) // 创建一个 edit 中的bot 实例、
	e.POST("/debug.step", DebugStep)     // 单步运行 edit 中的bot

	e.POST("/config.sys.info", ConfigGetSysInfo)
	e.POST("/config.sys.set", ConfigSetSysInfo)
	e.POST("/config.global.info", ConfigGetGlobalInfo)
	e.POST("/config.global.set", ConfigSetGlobalInfo)

	e.POST("/report.get", GetReport)
}
