package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/aosen/mlog"
	"github.com/aosen/robot"
	"gopkg.in/mgo.v2"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type MyProcessor struct {
}

func (this *MyProcessor) Process(p *robot.Page) {
	if !p.IsSucc() {
		mlog.LogInst().LogError(p.Errormsg())
		return
	}

	u, err := url.Parse(p.GetRequest().GetUrl())
	if err != nil {
		mlog.LogInst().LogError(err.Error())
		return
	}
	if !strings.HasSuffix(u.Host, "jiexieyin.org") {
		return
	}

	var urls []string
	query := p.GetHtmlParser()

	query.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		reJavascript := regexp.MustCompile("^javascript\\:")
		reLocal := regexp.MustCompile("^\\#")
		reMailto := regexp.MustCompile("^mailto\\:")
		if reJavascript.MatchString(href) || reLocal.MatchString(href) || reMailto.MatchString(href) {
			return
		}

		//处理相对路径
		var absHref string
		urlHref, err := url.Parse(href)
		if err != nil {
			mlog.LogInst().LogError(err.Error())
			return
		}
		if !urlHref.IsAbs() {
			urlPrefix := p.GetRequest().GetUrl()
			absHref = urlPrefix + href
			urls = append(urls, absHref)
		} else {
			urls = append(urls, href)
		}

	})

	p.AddTargetRequests(urls, "html")
	p.AddField("test1", p.GetRequest().GetUrl())
	p.AddField("test2", p.GetRequest().GetUrl())
}

func (this *MyProcessor) Finish() {

}

//mongo pipline 的例子，仅供参考，需要开发者自己实现
type PipelineMongo struct {
	session           *mgo.Session
	url               string
	mongoDBName       string //数据库名
	mongoDBCollection string //集合名
	c                 *mgo.Collection
	items             interface{} //存储的items结构体类型, 用于后期的反射
}

type Items struct {
	Test1 string `bson:"test1"`
	Test2 string `bson:"test2"`
}

func NewPipelineMongo(url, db, collection string) *PipelineMongo {
	session, err := mgo.Dial(url)
	if err != nil {
		panic("open mongodb file:" + err.Error())
	}
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	return &PipelineMongo{
		session:           session,
		url:               url,
		mongoDBName:       db,
		mongoDBCollection: collection,
		c:                 session.DB(db).C(collection),
	}
}

func (self *PipelineMongo) Process(pageitems *robot.PageItems, task robot.Task) {
	items := Items{}
	items.Test1, _ = pageitems.GetItem("test1")
	items.Test2, _ = pageitems.GetItem("test2")
	err := self.c.Insert(&items)
	if err != nil {
		panic(err)
	}
}
func main() {
	start_url := "http://www.jiexieyin.org"
	thread_num := uint(16)

	redisAddr := "127.0.0.1:6379"
	redisMaxConn := 10
	redisMaxIdle := 10

	mongoUrl := "localhost:27017"
	mongoDB := "test"
	mongoCollection := "test"

	proc := &MyProcessor{}

	sp := robot.NewSpider(proc, "redis_scheduler_example").
		//SetSleepTime("fixed", 6000, 6000).
		//SetScheduler(scheduler.NewQueueScheduler(true)).
		SetScheduler(robot.NewRedisScheduler(redisAddr, redisMaxConn, redisMaxIdle, true)).
		AddPipeline(robot.NewPipelineConsole()).
		AddPipeline(NewPipelineMongo(mongoUrl, mongoDB, mongoCollection)).
		SetThreadnum(thread_num)

	init := false
	for _, arg := range os.Args {
		if arg == "--init" {
			init = true
			break
		}
	}
	if init {
		sp.AddUrl(start_url, "html")
		mlog.LogInst().LogInfo("重新开始爬")
	} else {
		mlog.LogInst().LogInfo("继续爬")
	}
	sp.Run()
}
