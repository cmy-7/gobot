package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pojol/gobot/driver/behavior"
	"github.com/pojol/gobot/driver/bot"
	"github.com/pojol/gobot/driver/database"
	"github.com/pojol/gobot/driver/factory"
	"github.com/pojol/gobot/driver/utils"
)

type Response struct {
	Code int
	Msg  string
	Body interface{}
}

type Err int32

const (
	Succ Err = 200 + iota
	ErrContentRead
	ErrJsonInvalid
	ErrPluginLoad
	ErrMetaData
	ErrEnd
	ErrBreak
	ErrCantFindBot
	ErrCreateBot
	ErrEmptyBatch
)

var errmap map[Err]string = map[Err]string{
	ErrContentRead: "failed to read request content",
	ErrJsonInvalid: "wrong file format",
	ErrPluginLoad:  "failed to plugin load",
	ErrMetaData:    "failed to get meta data",
	ErrEnd:         "run to the end",
	ErrBreak:       "run to the break",
	ErrCantFindBot: "can't find bot",
	ErrCreateBot:   "failed to create bot, the behavior tree file needs to be uploaded to the server before creation",
	ErrEmptyBatch:  "empty batch info",
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

	_, err = behavior.New(bts)
	if err != nil {
		fmt.Println(err.Error())
		code = ErrJsonInvalid
		goto EXT
	}
	factory.Global.AddBehavior(name, bts)

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
	if upload.GetFileExt() != ".xml" {
		code = ErrJsonInvalid
		goto EXT
	}
	fbyte = upload.ReadBytes()
	if len(fbyte) == 0 {
		code = ErrContentRead
		goto EXT
	}

	name = upload.FileName()
	_, err = behavior.New(fbyte)
	if err != nil {
		fmt.Println(err.Error())
		code = ErrJsonInvalid
		goto EXT
	}
	factory.Global.AddBehavior(name, fbyte)

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]

	ctx.JSON(http.StatusOK, res)
	return nil
}

type behaviorInfo struct {
	Name   string
	Update int64
}
type BehaviorListRes struct {
	Bots []behaviorInfo
}

type FileRemoveReq struct {
	Name string
}
type FileRemoveRes struct {
	Bots []behaviorInfo
}

func FileRemove(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	code := Succ
	res := &Response{}
	req := &FileRemoveReq{}
	body := &FileRemoveRes{}

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

	factory.Global.RmvBehavior(req.Name)
	for _, v := range factory.Global.GetBehaviors() {
		body.Bots = append(body.Bots, behaviorInfo{
			Name:   v.Name,
			Update: v.UpdateTime,
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
	code := Succ
	res := &Response{}
	body := &BehaviorListRes{}

	info := factory.Global.GetBehaviors()
	for _, v := range info {
		body.Bots = append(body.Bots, behaviorInfo{
			Name:   v.Name,
			Update: v.UpdateTime,
		})
	}

	res.Code = int(code)
	res.Msg = errmap[code]
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

type FindBehaviorReq struct {
	Name string
}

type FindBehaviorRes struct {
	Info database.BehaviorInfo
}

func FileGetBlob(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	req := &FindBehaviorReq{}
	info := database.BehaviorInfo{}

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

	info, err = factory.Global.FindBehavior(req.Name)
	if err != nil {
		fmt.Println(err.Error())
		goto EXT
	}

EXT:
	ctx.Blob(http.StatusOK, "text/plain;charset=utf-8", info.Dat)
	return nil
}

type ReportApiInfo struct {
	Api        string
	ReqNum     int
	ErrNum     int
	ConsumeNum int64

	ReqSize int64
	ResSize int64
}

type ReportInfo struct {
	ID        string
	Name      string
	BotNum    int
	ReqNum    int
	ErrNum    int
	Tps       int
	Dura      string
	BeginTime string
	Apilst    []ReportApiInfo
}

type ReportRes struct {
	Info []ReportInfo
}

func GetReport(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	body := &ReportRes{}

	rep := factory.Global.GetReport()
	for _, v := range rep {
		info := ReportInfo{
			ID:        v.ID,
			Name:      v.Name,
			BotNum:    v.BotNum,
			ReqNum:    v.ReqNum,
			ErrNum:    v.ErrNum,
			Tps:       v.Tps,
			Dura:      v.Dura,
			BeginTime: v.BeginTime.Format("2006-01-02 15:04:05"),
		}

		for api, detail := range v.UrlMap {
			info.Apilst = append(info.Apilst, ReportApiInfo{
				Api:        api,
				ReqNum:     detail.ReqNum,
				ConsumeNum: detail.AvgNum,
				ReqSize:    detail.ReqSize,
				ResSize:    detail.ResSize,
				ErrNum:     detail.ErrNum,
			})
		}
		body.Info = append(body.Info, info)
	}

	res.Code = int(Succ)
	res.Msg = ""
	res.Body = body

	ctx.JSON(http.StatusOK, res)
	return nil
}

type RunRequest struct {
	Name string
	Num  int
}

type RunResponse struct {
}

func BotRun(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	req := &RunRequest{}
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

	fmt.Println("add task", req.Name)
	err = factory.Global.AddTask(req.Name, int32(req.Num))
	if err != nil {
		res.Msg = err.Error()
	}

EXT:
	res.Code = int(code)
	res.Msg = errmap[code]

	ctx.JSON(http.StatusOK, res)
	return nil
}

type BotListResponse struct {
	Lst []factory.BatchInfo
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

type StepRequest struct {
	BotID string
}

type StepResponse struct {
	Prev       string
	Cur        string
	Blackboard string
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

	s = b.RunStep()
	body.Blackboard, err = b.GetMetadata()
	fmt.Println("blackboard", body.Blackboard, err)
	body.Cur = b.GetCurNodeID()
	body.Prev = b.GetPrevNodeID()

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

type createDebugBotResponse struct {
	BotID string
}

func DebugCreate(ctx echo.Context) error {
	ctx.Response().Header().Set("Access-Control-Allow-Origin", "*")
	res := &Response{}
	body := &createDebugBotResponse{}
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

	//e.Use(ReqPrint())

	e.POST("/file.txtUpload", FileTextUpload)
	e.POST("/file.blobUpload", FileBlobUpload)
	e.POST("/file.remove", FileRemove)

	e.POST("/file.list", FileGetList)
	e.POST("/file.get", FileGetBlob)

	e.POST("/bot.create", BotRun) // 创建一批bot
	e.POST("/bot.list", BotList)
	//e.POST("/bot.list")
	//e.POST("/bot.info")	// 获取所有运行时的bot信息（保留100个  运行中 | 已终止 | 有错误

	e.POST("/debug.create", DebugCreate) // 创建一个 edit 中的bot 实例、
	e.POST("/debug.step", DebugStep)     // 单步运行 edit 中的bot

	e.POST("/get.report", GetReport)
}