package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Akegarasu/blive-queue/eio"
	bliveApi "github.com/Akegarasu/blivedm-go/api"
	bliveClient "github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"

	//"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	Eio           *eio.Server
	DanmakuClient *bliveClient.Client
	Queue         *Queue
	Rule          *Rule
	RoomID        string
	Pause         bool
	Cookie        string
}

func NewServer() *Server {
	return &Server{
		Eio:           eio.NewServer(),
		DanmakuClient: nil,
		Queue:         NewQueue("0"),
		Rule:          DefaultRule(),
		Pause:         false,
	}
}

func (s *Server) Init() {
	s.Eio.RegisterEventHandler("HEARTBEAT", func(event *eio.Event) {
		log.Debug("heartbeat")
	})

	s.Eio.RegisterEventHandler("CONNECT_DANMAKU", func(event *eio.Event) {
		var payload struct {
			RoomID string `json:"roomId"`
			Cookie string `json:"cookie"`
		}
		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			log.Error("CONNECT_DANMAKU参数解析失败: ", err)
			return
		}
		s.ConnectDanmakuServer(payload.RoomID, payload.Cookie)
	})

	s.Eio.RegisterEventHandler("APPLY_RULE", func(event *eio.Event) {
		s.Rule = NewRule(event.Data)
		log.Infof("设置了新的过滤规则 关键词: %s, 最大人数: %d, 仅舰长: %v, 最低牌子等级: %d", s.Rule.keyword, s.Rule.maxQueueLength, s.Rule.guardOnly, s.Rule.minMedalLevel)
	})

	s.Eio.RegisterEventHandler("ADD_USER", func(event *eio.Event) {
		_ = s.Eio.BoardCastEvent(*event)
		log.Debug("测试用户: ", event.Data)
	})

	s.Eio.RegisterEventHandler("REMOVE_USER", func(event *eio.Event) {
		uid, err := strconv.Atoi(event.Data)
		if err != nil {
			log.Error("删除失败: uid转换出错")
		}
		ok := s.Queue.Remove(uid)
		if !ok {
			log.Error("删除失败啦")
			return
		}
		_ = s.Eio.BoardCastEvent(*event)
		log.Info("删除了uid: ", event.Data)
	})

	s.Eio.RegisterEventHandler("REMOVE_ALL", func(event *eio.Event) {
		s.Queue.Clear()
		_ = s.Eio.BoardCastEvent(*event)
		log.Info("清空了排队")
	})

	s.Eio.RegisterEventHandler("TOP_USER", func(event *eio.Event) {
		uid, err := strconv.Atoi(event.Data)
		if err != nil {
			log.Error("置顶失败: uid转换出错")
		}
		ok := s.Queue.Top(uid)
		if !ok {
			log.Error("置顶失败啦")
			return
		}
		_ = s.Eio.BoardCastEvent(*event)
		log.Debug("置顶了uid: ", event.Data)
	})

	s.Eio.RegisterEventHandler("START_USER", func(event *eio.Event) {
		uid, err := strconv.Atoi(event.Data)
		if err != nil {
			log.Error("开始失败: uid转换出错")
		}
		ok := s.Queue.Start(uid)
		if !ok {
			log.Error("开始失败啦")
			return
		}
		_ = s.Eio.BoardCastEvent(*event)
		log.Debug("开始了uid: ", event.Data)
	})

	/*
		s.Eio.RegisterEventHandler("RESORT", func(event *eio.Event) {
			j := gjson.Parse(event.Data)
			oldIndex := int(j.Get("oldIndex").Int())
			newIndex := int(j.Get("newIndex").Int())
			s.Queue.Resort(oldIndex, newIndex)
			_ = s.Eio.BoardCastEventExceptSelf(*event)
			log.Infof("排序: %d -> %d", oldIndex, newIndex)
		})
	*/

	s.Eio.RegisterEventHandler("PAUSE", func(event *eio.Event) {
		s.Pause = true
		log.Info("已暂停排队")
	})

	s.Eio.RegisterEventHandler("CONTINUE", func(event *eio.Event) {
		s.Pause = false
		log.Info("已继续排队")
	})
}

func (s *Server) ConnectDanmakuServer(roomID string, cookie string) {
	if s.DanmakuClient != nil {
		s.DanmakuClient.Stop()
	}
	rid, err := strconv.Atoi(roomID)
	if err != nil {
		log.Error("房间 ID 解析错误")
	}
	c := bliveClient.NewClient(rid)
	c.SetCookie(cookie)
	c.OnDanmaku(s.HandleDanmaku)
	c.OnGift(s.HandleGiftJoinQueue)
	c.OnGuardBuy(s.HandleNewGuardJoinQueue)
	c.OnLiveStop(s.HandleLiveStop)
	//c.OnLiveStart(s.HandleLiveStart)
	err = c.Start()
	if err != nil {
		log.Warn("连接弹幕服务器出错")
	}
	s.DanmakuClient = c
	s.RoomID = roomID
	log.Info("连接到房间: ", roomID)
	s.Queue = NewQueue(roomID)
}

// HandleDanmaku 处理弹幕，弹幕的原始数据应只停留在这个函数。往下传的参数全部应该为 message.User
func (s *Server) HandleDanmaku(d *message.Danmaku) {
	if s.Pause {
		return
	}
	if dev {
		log.Info("弹幕内容: ", d.Content)
	}
	if s.Rule.fuzzyMatch {
		if strings.Contains(d.Content, s.Rule.cancelKeyword) {
			s.HandleLeaveQueue(d.Sender)
		} else if strings.Contains(d.Content, s.Rule.keyword) {
			s.HandleJoinQueue(d.Sender)
		}
	} else {
		switch d.Content {
		case s.Rule.cancelKeyword:
			s.HandleLeaveQueue(d.Sender)
		case s.Rule.keyword:
			s.HandleJoinQueue(d.Sender)
		}
	}
}

func (s *Server) HandleJoinQueue(user *message.User) {
	if !s.Rule.Filter(user, s.RoomID) {
		return
	}
	users, _ := s.Queue.FetchOrderedQueue()
	if len(users) >= s.Rule.maxQueueLength {
		log.Error("排队失败: 队列满了")
		return
	}
	u := QueueUser{
		Uid:        user.Uid,
		Uname:      user.Uname,
		GuardLevel: user.GuardLevel,
		Gifts:      0,
		Now:        0,
	}
	if ok := s.Queue.Add(&u); ok {

		log.Infof("添加排队成功: %s (uid: %d)", user.Uname, user.Uid)
		err := s.Eio.BoardCastEvent(eio.Event{
			EventName: "ADD_USER",
			Data:      NewLiveUser(&u).Json(),
		})
		if err != nil {
			log.Error("同步排队事件失败: 请尝试在控制台手动点击 “同步” 按钮")
		}
	} else {
		log.Errorf("排队失败: %s (uid: %d) 已经在队列里面了", user.Uname, user.Uid)
	}
}

// 好吧，毕竟礼物不算弹幕，只能再处理一次源数据了
func (s *Server) HandleGiftJoinQueue(gift *message.Gift) {
	if gift.CoinType != "gold" {
		return
	}
	ms := message.User{
		Uid:        gift.Uid,
		Uname:      gift.Uname,
		GuardLevel: gift.GuardLevel,
		Medal: &message.Medal{
			Name:     gift.MedalInfo.MedalName,
			Level:    gift.MedalInfo.MedalLevel,
			UpRoomId: gift.MedalInfo.AnchorRoomid,
		},
	}
	if !s.Rule.Filter(&ms, s.RoomID) {
		return
	}
	gft := gift.Price * gift.Num / 100
	u := QueueUser{
		Uid:        gift.Uid,
		Uname:      gift.Uname,
		GuardLevel: gift.GuardLevel,
		Gifts:      gft,
		Now:        0,
	}
	if gft >= 52 {
		if ok := s.Queue.Add(&u); ok {
			log.Infof("添加排队成功: %s (uid: %d)", gift.Uname, gift.Uid)
			err := s.Eio.BoardCastEvent(eio.Event{
				EventName: "ADD_USER",
				Data:      NewLiveUser(&u).Json(),
			})
			if err != nil {
				log.Error("同步排队事件失败: 请尝试在控制台手动点击 “同步” 按钮")
			} else {
				log.Errorf("排队失败: %s (uid: %d) 已经在队列里面了", gift.Uname, gift.Uid)
			}
		}
	} else {
		if ok := s.Queue.UpdateGifts(&u); ok {
			log.Infof("更新成功: %s (uid: %d) 的礼物电池数 %d 计入了", gift.Uname, gift.Uid, gft)
		}
	}
}

// 好吧，毕竟舰长也不算弹幕，只能再处理一次源数据了
func (s *Server) HandleNewGuardJoinQueue(guard *message.GuardBuy) {
	/*
		ms := message.User{
			Uid:        guard.Uid,
			Uname:      guard.Username,
			GuardLevel: guard.GuardLevel,
		}
		if !s.Rule.Filter(&ms, s.RoomID) {
			return
		}
	*/
	var level int
	switch guard.GuardLevel {
	case 3:
		level = 95
	case 2:
		level = 98
	case 1:
		level = 99
	default:
		level = 0
	}
	u := QueueUser{
		Uid:        guard.Uid,
		Uname:      guard.Username,
		GuardLevel: level,
		Gifts:      0,
		Now:        0,
	}
	if ok := s.Queue.Add(&u); ok {
		log.Infof("添加排队成功: %s (uid: %d)", guard.Username, guard.Uid)
		err := s.Eio.BoardCastEvent(eio.Event{
			EventName: "ADD_USER",
			Data:      NewLiveUser(&u).Json(),
		})
		if err != nil {
			log.Error("同步排队事件失败: 请尝试在控制台手动点击 “同步” 按钮")
		} else {
			log.Errorf("排队失败: %s (uid: %d) 已经在队列里面了", guard.Username, guard.Uid)
		}
	}
}

func (s *Server) HandleLeaveQueue(user *message.User) {
	if ok := s.Queue.Remove(user.Uid); ok {
		_ = s.Eio.BoardCastEvent(eio.Event{
			EventName: "REMOVE_USER",
			Data:      strconv.Itoa(user.Uid),
		})
		log.Infof("取消排队成功: %s (uid: %d)", user.Uname, user.Uid)
	} else {
		log.Errorf("取消排队失败: %s (uid: %d) 根本没有排队哦", user.Uname, user.Uid)
	}
}

func (s *Server) HandleLiveStop(m *message.LiveStop) {
	log.Info("直播结束了，清空排队并保存未完成的插队列表")
	s.Queue.formCutinAndClearQueue()
	s.Queue.ClearTotalGifts()
	err := s.Eio.BoardCastEvent(eio.Event{
		EventName: "REMOVE_ALL",
		Data:      "",
	})
	if err != nil {
		log.Error("同步清空排队事件失败: 请尝试在控制台手动点击 “同步” 按钮")
	}
}

func (s *Server) HandleLiveStart(m *message.LiveStart) {
	log.Info("直播开始了，初始化队列")
	s.Queue.Clear()
	s.Queue.ClearTotalGifts()
	err := s.Eio.BoardCastEvent(eio.Event{
		EventName: "REMOVE_ALL",
		Data:      "",
	})
	if err != nil {
		log.Error("同步清空排队事件失败: 请尝试在控制台手动点击 “同步” 按钮")
	}
}

func (s *Server) sendDanmaku(msg string) error {
	d, err := bliveApi.SendDefaultDanmaku(s.RoomID, msg, &bliveApi.BiliVerify{
		Csrf:     "",
		SessData: "",
	})
	if err != nil {
		log.Error("弹幕发送失败")
		return err
	}
	log.Infof("发送了弹幕: %s", d.Msg)
	return nil
}
