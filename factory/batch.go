package factory

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/pojol/gobot/bot"
	"github.com/pojol/gobot/bot/behavior"
	"github.com/pojol/gobot/database"
	"github.com/pojol/gobot/utils"
)

type BatchInfo struct {
	ID     string
	Name   string
	Cur    int32
	Max    int32
	Errors int32
}

type Batch struct {
	ID           string
	Name         string
	cursorNum    int32
	CurNum       int32
	TotalNum     int32
	BatchNum     int32
	enqueneDelay int32
	Errors       int32

	treeData     []byte
	path         string
	globalScript string

	bots    map[string]*bot.Bot
	colorer *color.Color
	rep     *database.ReportDetail

	bwg  utils.SizeWaitGroup
	exit *utils.Switch

	pipeline  chan *bot.Bot
	done      chan interface{}
	BatchDone chan interface{}

	botDoneCh chan string
	botErrCh  chan bot.ErrInfo
}

type BatchConfig struct {
	batchsize     int32
	globalScript  string
	scriptPath    string
	enqeueneDelay int32
}

func CreateBatch(name string, cur, total int32, tbyt []byte, cfg BatchConfig) *Batch {

	b := &Batch{
		ID:           uuid.New().String(),
		Name:         name,
		path:         cfg.scriptPath,
		globalScript: cfg.globalScript,
		enqueneDelay: cfg.enqeueneDelay,
		CurNum:       cur,
		BatchNum:     cfg.batchsize,
		TotalNum:     total,
		bwg:          utils.NewSizeWaitGroup(int(cfg.batchsize)),
		exit:         utils.NewSwitch(),
		treeData:     tbyt,
		pipeline:     make(chan *bot.Bot, cfg.batchsize),
		done:         make(chan interface{}, 1),
		BatchDone:    make(chan interface{}, 1),
		botDoneCh:    make(chan string),
		botErrCh:     make(chan bot.ErrInfo),

		colorer: color.New(),
		bots:    make(map[string]*bot.Bot),
	}

	fmt.Println("create", total, "bot", "pipeline size", cfg.batchsize)
	database.GetTask().New(database.TaskTable{
		ID:          b.ID,
		Name:        name,
		TotalNumber: b.TotalNum,
		CurNumber:   0,
	})

	go b.loop()
	b.run()

	return b
}

func (b *Batch) Info() BatchInfo {
	cur := atomic.LoadInt32(&b.CurNum)

	return BatchInfo{
		ID:     b.ID,
		Name:   b.Name,
		Cur:    cur,
		Max:    b.TotalNum,
		Errors: atomic.LoadInt32(&b.Errors),
	}
}

func (b *Batch) Report() database.ReportDetail {
	return *b.rep
}

func (b *Batch) push(bot *bot.Bot) {
	fmt.Println("bot", bot.ID(), "push", atomic.LoadInt32(&b.cursorNum), "=>", b.TotalNum)

	b.bots[bot.ID()] = bot
}

func (b *Batch) pop(id string) {
	b.bwg.Done()
	atomic.AddInt32(&b.CurNum, 1)

	fmt.Println("bot", id, "pop", atomic.LoadInt32(&b.CurNum), "=>", b.TotalNum)
}

func (b *Batch) loop() {

	b.rep = &database.ReportDetail{
		ID:        b.ID,
		Name:      b.Name,
		BeginTime: time.Now(),
		UrlMap:    make(map[string]*database.ApiDetail),
	}

	for {
		select {
		case botptr := <-b.pipeline:
			b.push(botptr)
			botptr.RunByThread(b.botDoneCh, b.botErrCh)
		case id := <-b.botDoneCh:
			if _, ok := b.bots[id]; ok {
				b.pushReport(b.rep, b.bots[id])
			}
			b.pop(id)
		case err := <-b.botErrCh:
			if _, ok := b.bots[err.ID]; ok {
				b.pushReport(b.rep, b.bots[err.ID])
			}
			atomic.AddInt32(&b.Errors, 1)
			b.pop(err.ID)
		case <-b.done:
			goto ext
		}
	}
ext:
	b.record()
	b.exit.Done()
	b.BatchDone <- 1
}

func (b *Batch) run() {

	go func() {

		for {

			if b.exit.HasOpend() {
				fmt.Println("break running")
				break
			}

			var curbatchnum int32
			last := b.TotalNum - atomic.LoadInt32(&b.CurNum)
			if b.BatchNum < last {
				curbatchnum = b.BatchNum
			} else {
				curbatchnum = last
			}

			fmt.Println("batch", b.ID, "begin size =", curbatchnum)
			for i := 0; i < int(curbatchnum); i++ {
				atomic.AddInt32(&b.cursorNum, 1)
				b.bwg.Add()

				tree, _ := behavior.Load(b.treeData, behavior.Thread)
				b.pipeline <- bot.NewWithBehaviorTree(b.path, tree, b.Name, b.ID, atomic.LoadInt32(&b.cursorNum), b.globalScript)
				time.Sleep(time.Millisecond * time.Duration(b.enqueneDelay))
			}

			b.bwg.Wait()
			database.GetTask().Update(b.ID, atomic.LoadInt32(&b.CurNum))
			fmt.Println("batch", b.ID, "end", atomic.LoadInt32(&b.CurNum), "=>", b.TotalNum)
			if atomic.LoadInt32(&b.CurNum) >= b.TotalNum {
				b.done <- 1
			}

			time.Sleep(time.Millisecond * 100)
		}

	}()

}

func (b *Batch) Close() {

}

func (b *Batch) pushReport(rep *database.ReportDetail, bot *bot.Bot) {
	rep.BotNum++
	robotReport := bot.GetReport()

	rep.ReqNum += len(robotReport)
	for _, v := range robotReport {
		if _, ok := rep.UrlMap[v.Api]; !ok {
			rep.UrlMap[v.Api] = &database.ApiDetail{}
		}

		rep.UrlMap[v.Api].ReqNum++
		rep.UrlMap[v.Api].AvgNum += int64(v.Consume)
		rep.UrlMap[v.Api].ReqSize += int64(v.ReqBody)
		rep.UrlMap[v.Api].ResSize += int64(v.ResBody)
		if v.Err != "" {
			rep.ErrNum++
			rep.UrlMap[v.Api].ErrNum++
		}
	}

}

func (b *Batch) record() {

	fmt.Println("+--------------------------------------------------------------------------------------------------------+")
	fmt.Printf("Req url%-33s Req count %-5s Average time %-5s Body req/res %-5s Succ rate %-10s\n", "", "", "", "", "")

	arr := []string{}
	for k := range b.rep.UrlMap {
		arr = append(arr, k)
	}
	sort.Strings(arr)

	var reqtotal int64

	for _, sk := range arr {
		v := b.rep.UrlMap[sk]
		var avg string
		if v.AvgNum == 0 {
			avg = "0 ms"
		} else {
			avg = strconv.Itoa(int(v.AvgNum/int64(v.ReqNum))) + "ms"
		}

		succ := strconv.Itoa(v.ReqNum-v.ErrNum) + "/" + strconv.Itoa(v.ReqNum)

		reqsize := strconv.Itoa(int(v.ReqSize/1024)) + "kb"
		ressize := strconv.Itoa(int(v.ResSize/1024)) + "kb"

		reqtotal += int64(v.ReqNum)

		u, _ := url.Parse(sk)
		if v.ErrNum != 0 {
			b.colorer.Printf("%-40s %-15d %-18s %-18s %-10s\n", u.Path, v.ReqNum, avg, reqsize+" / "+ressize, utils.Red(succ))
		} else {
			fmt.Printf("%-40s %-15d %-18s %-18s %-10s\n", u.Path, v.ReqNum, avg, reqsize+" / "+ressize, succ)
		}
	}
	fmt.Println("+--------------------------------------------------------------------------------------------------------+")

	durations := int(time.Since(b.rep.BeginTime).Seconds())
	if durations <= 0 {
		durations = 1
	}

	qps := int(reqtotal / int64(durations))
	duration := strconv.Itoa(durations) + "s"

	b.rep.Tps = qps
	b.rep.Dura = duration

	if b.rep.ErrNum != 0 {
		b.colorer.Printf("robot : %d match to %d APIs req count : %d duration : %s qps : %d errors : %v\n", b.rep.BotNum, len(b.rep.UrlMap), b.rep.ReqNum, duration, qps, utils.Red(b.rep.ErrNum))
	} else {
		fmt.Printf("robot : %d match to %d APIs req count : %d duration : %s qps : %d errors : %d\n", b.rep.BotNum, len(b.rep.UrlMap), b.rep.ReqNum, duration, qps, b.rep.ErrNum)
	}

}
