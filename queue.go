package main

import (
	"database/sql"
	"encoding/json"
	"sync"

	//"github.com/Akegarasu/blivedm-go/message"
	_ "github.com/mattn/go-sqlite3"

	log "github.com/sirupsen/logrus"
)

type Queue struct {
	mu     sync.RWMutex
	roomID string
}

type SyncMessage struct {
	Cmd  string     `json:"cmd"`
	Data []LiveUser `json:"data"`
}

type QueueUser struct {
	Uid        int
	Uname      string
	GuardLevel int
	Gifts      int
	Now        int
}

type LiveUser struct {
	Uid        string `json:"uid"`
	Nickname   string `json:"nickname"`
	GuardLevel string `json:"level"`
	Gifts      string `json:"gifts"`
	Now        string `json:"now"`
}

func NewLiveUser(u *QueueUser) LiveUser {
	return LiveUser{
		Uid:        i2s(u.Uid),
		Nickname:   u.Uname,
		GuardLevel: i2s(u.GuardLevel),
		Gifts:      i2s(u.Gifts),
		Now:        i2s(u.Now),
	}
}

func (b LiveUser) Json() string {
	marshal, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(marshal)
}

func NewQueue(roomID string) *Queue {
	db, err := sql.Open("sqlite3", "file:queue_"+roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		log.Error("打开数据库失败: ", err)
		// Don't return nil, return an empty Queue
	}
	defer db.Close()

	db.Exec(`CREATE TABLE IF NOT EXISTS queue (
        uid INTEGER PRIMARY KEY,
        nickname TEXT,
        level INTEGER,
        gifts INTEGER,
        topped INTEGER DEFAULT 0,
		now INTEGER DEFAULT 0,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );`)
	db.Exec(`CREATE TABLE IF NOT EXISTS totalGifts (
        uid INTEGER PRIMARY KEY,
        nickname TEXT,
        level INTEGER,
        gifts INTEGER
    );`)
	db.Exec(`CREATE TABLE IF NOT EXISTS lastCutin (
        uid INTEGER PRIMARY KEY
    );`)
	return &Queue{
		roomID: roomID,
	}
}

func (q *Queue) Add(u *QueueUser) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q._Add(u)
}

func (q *Queue) _Add(u *QueueUser) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	if q.isNow(u.Uid) {
		return false
	}

	rows, err := db.Query("SELECT level, gifts FROM queue WHERE uid = ?", u.Uid)
	if err != nil {
		return false
	}
	defer rows.Close()

	gf := u.Gifts
	if rows.Next() {
		var level, gifts int
		if err := rows.Scan(&level, &gifts); err != nil {
			return false
		}
		err = sql.ErrNoRows
		if u.GuardLevel > 80 {
			if level < u.GuardLevel {
				_, err = db.Exec("UPDATE queue SET level = ?, timestamp = CURRENT_TIMESTAMP WHERE uid = ?", u.GuardLevel, u.Uid)
				if err != nil {
					return false
				}
			}
		} else if gf > 0 {
			gf = gifts + u.Gifts
			_, err = db.Exec("UPDATE queue SET gifts = ? WHERE uid = ?", gf, u.Uid)
			if err != nil {
				return false
			}
		}
	} else {
		if u.GuardLevel > 80 {
			gf = 0
		}
		var gl int
		switch u.GuardLevel {
		case 1:
			gl = 3
		case 3:
			gl = 1
		default:
			gl = u.GuardLevel
		}
		gf_old := 0
		err = db.QueryRow("SELECT gifts FROM totalGifts WHERE uid = ?", u.Uid).Scan(&gf_old)
		if err != nil {
			gf_old = 0
		}
		gf += gf_old
		_, err = db.Exec("INSERT INTO queue (uid, nickname, level, gifts) VALUES (?, ?, ?, ?)", u.Uid, u.Uname, gl, gf)
		if err != nil {
			return false
		}
		db.Exec("DELETE FROM totalGifts WHERE uid = ?", u.Uid)
	}
	if q.isCutin(u.Uid) {
		_, err = db.Exec("UPDATE queue SET topped = 65536 WHERE uid = ?", u.Uid)
		if err != nil {
			return false
		}
		db.Exec("DELETE FROM lastCutin WHERE uid = ?", u.Uid)
	}
	return err == nil
}

func (q *Queue) Remove(uid int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM queue WHERE uid = ?", uid)
	return err == nil
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return
	}
	defer db.Close()

	db.Exec("DELETE FROM queue")
}

func (q *Queue) ClearTotalGifts() {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return
	}
	defer db.Close()

	db.Exec("DELETE FROM total_gifts")
}

func (q *Queue) Top(uid int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q._Top(uid)
}

func (q *Queue) _Top(uid int) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	var topped int
	err = db.QueryRow("SELECT topped FROM queue WHERE now = 0 AND topped < 65535 ORDER BY topped DESC LIMIT 1").Scan(&topped)
	if err != nil {
		return false
	}
	_, err = db.Exec("UPDATE queue SET topped = ? WHERE now = 0 AND uid = ?", topped+1, uid)
	return err == nil
}

func (q *Queue) FetchOrderedQueue() ([]*QueueUser, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	users := make(map[int]QueueUser)
	var uidList []int

	querySet := [...]string{
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 1",                                                                                      // 0. NOW
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped > 0 ORDER BY topped DESC, level DESC, gifts DESC, timestamp ASC",           // 1. Topped users
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level > 0 ORDER BY level DESC, timestamp ASC",                      // 2. New & old abos
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts >= 52 ORDER BY gifts DESC, timestamp ASC",      // 3. Cut-ins
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts < 52 AND timestamp > 0 ORDER BY timestamp ASC", // 4. All users
	}

	for i := 0; i < len(querySet); i++ {
		rows, err := db.Query(querySet[i])
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var uid, level, gifts, now int
			var nickname string
			if err := rows.Scan(&uid, &nickname, &level, &gifts, &now); err != nil {
				continue
			}
			users[uid] = QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        now,
			}
			uidList = append(uidList, uid)
		}
	}

	var result []*QueueUser
	for _, uid := range uidList {
		user := users[uid]
		result = append(result, &user)
	}
	return result, nil
}

func (q *Queue) FetchFirstUser() *QueueUser {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q._FetchUser(0)
}

func (q *Queue) _FetchUser(pos int) *QueueUser {
	if pos < 0 {
		return nil
	}
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return nil
	}
	defer db.Close()

	var uid, level, gifts, now int
	var nickname string
	flag := false
	_cnt := 0

	querySet := [...]string{
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 1",                                                                                      // 0. NOW
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped > 0 ORDER BY topped DESC, level DESC, gifts DESC, timestamp ASC",           // 1. Topped users
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level > 0 ORDER BY level DESC, timestamp ASC",                      // 2. New & old abos
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts >= 52 ORDER BY gifts DESC, timestamp ASC",      // 3. Cut-ins
		"SELECT uid, nickname, level, gifts, now FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts < 52 AND timestamp > 0 ORDER BY timestamp ASC", // 4. All users
	}

	for i := 0; i < len(querySet); i++ {
		rows, err := db.Query(querySet[i])
		if err != nil {
			return nil
		}
		defer rows.Close()

		for rows.Next() {
			if err := rows.Scan(&uid, &nickname, &level, &gifts, &now); err != nil {
				continue
			}
			if _cnt < pos {
				_cnt++
				continue
			} else {
				flag = true
				break
			}
		}
		if flag {
			break
		}
	}

	if flag {
		return &QueueUser{
			Uid:        uid,
			Uname:      nickname,
			GuardLevel: level,
			Gifts:      gifts,
			Now:        now,
		}
	}

	return nil
}

func (q *Queue) UpdateGifts(u *QueueUser) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	if q.isNow(u.Uid) {
		return false
	}

	db.Exec("INSERT INTO totalGifts VALUES (?, ?, ?, ?) ON CONFLICT(uid) DO UPDATE SET gifts = gifts + ?", u.Uid, u.Uname, u.GuardLevel, u.Gifts, u.Gifts)
	var gft int
	err = db.QueryRow("SELECT gifts FROM totalGifts WHERE uid = ?", u.Uid).Scan(&gft)
	if err != nil {
		return false
	}
	if gft >= 52 {
		ok := q._Add(&QueueUser{
			Uid:        u.Uid,
			Uname:      u.Uname,
			GuardLevel: u.GuardLevel,
			Gifts:      gft,
			Now:        0,
		})
		if ok {
			_, err = db.Exec("DELETE FROM totalGifts WHERE uid = ?", u.Uid)
			return err == nil
		} else {
			return false
		}
	}
	return true
}

func (q *Queue) In(u *QueueUser) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	var exists int
	err = db.QueryRow("SELECT 1 FROM queue WHERE uid = ?", u.Uid).Scan(&exists)
	return err == nil
}

func (q *Queue) Encode() *SyncMessage {
	users, err := q.FetchOrderedQueue()
	if err != nil {
		return &SyncMessage{Cmd: "SYNC", Data: []LiveUser{}}
	}
	d := make([]LiveUser, 0, len(users))
	for _, user := range users {
		d = append(d, NewLiveUser(user))
	}
	return &SyncMessage{
		Cmd:  "SYNC",
		Data: d,
	}
}

func (q *Queue) formCutin() {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT uid FROM queue WHERE now = 0 AND gifts >= 52")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, err = db.Exec("INSERT INTO lastCutin (uid) VALUES (?)", uid)
		if err != nil {
			continue
		}
	}
}

func (q *Queue) formCutinAndClearQueue() {
	q.formCutin()
	q.Clear()
}

func (q *Queue) clearCutin() {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return
	}
	defer db.Close()

	db.Exec("DELETE FROM lastCutin")
}

func (q *Queue) isCutin(uid int) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	var exists int
	err = db.QueryRow("SELECT 1 FROM lastCutin WHERE uid = ?", uid).Scan(&exists)
	if err != nil {
		return false
	}
	return exists > 0
}

func (q *Queue) isNow(uid int) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	exists := 0
	err = db.QueryRow("SELECT 1 FROM queue WHERE uid = ? AND now = 1", uid).Scan(&exists)
	if err != nil {
		return false
	}
	return exists > 0
}

func (q *Queue) Start(uid int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q._Start(uid)
}

func (q *Queue) _Start(uid int) bool {
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM queue WHERE now = 1")
	if err != nil {
		return false
	}
	_, err = db.Exec("UPDATE queue SET now = 1 WHERE uid = ?", uid)
	return err == nil
}

func (q *Queue) NextUser() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	u := q._FetchUser(0)
	if u != nil {
		if u.Now == 1 {
			u = q._FetchUser(1)
			if u == nil {
				return false
			}
		}
		return q._Start(u.Uid)
	}
	return false
}
