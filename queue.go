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
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	if q.isNow(u.Uid) {
		return false
	}

	rows, err := db.Query("SELECT uid FROM queue WHERE uid = ?", u.Uid)
	if err != nil {
		return false
	}
	defer rows.Close()

	gf := u.Gifts
	if rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
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
	/*
		// Get a list of users in the queue
		list, err := q.FetchOrderedQueue()
		if err != nil {
			return false
	*/

	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	/*
		// Check if someone is inserted before the removed user
		for i, user := range list {
			if user.Uid == uid {
				// If we found the user, check if the one before was inserted using before_uid
				if i > 0 {
					var exists int
					err = db.QueryRow("SELECT 1 FROM queue WHERE uid = ? AND before_uid = ?", list[i-1].Uid, list[i].Uid).Scan(&exists)
					if err != nil {
						break
					}
					if exists > 0 {
						var nextUid int
						if i+1 < len(list) {
							nextUid = list[i+1].Uid
						} else {
							nextUid = 0
						}
						_, err = db.Exec("UPDATE queue SET before_uid = ? WHERE uid = ?", nextUid, list[i-1].Uid)
						if err != nil {
							break
						}
					}
				}
				break
			}
		}
	*/

	// Now you can safely remove this user without ruining the whole queue
	_, err = db.Exec("DELETE FROM queue WHERE uid = ?", uid)
	return err == nil
}

/*
func (q *Queue) Resort(oldIndex int, newIndex int) bool {
	if oldIndex == newIndex {
		return false
	}
	list, err := q.FetchOrderedQueue()
	if err != nil {
		return false
	}
	if oldIndex < 0 || oldIndex >= len(list) || newIndex < 0 || newIndex >= len(list) {
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	db, err := sql.Open("sqlite3", "file:queue_"+q.roomID+".db?cache=shared&mode=rwc")
	if err != nil {
		return false
	}
	defer db.Close()

	var beforeUid int

	// Determine the new beforeUid for the moved item
	if newIndex+1 < len(list) {
		beforeUid = list[newIndex].Uid
	} else {
		beforeUid = -1
	}
	// Update the before_uid of the moved item
	_, err = db.Exec("UPDATE queue SET before_uid = ? WHERE uid = ?", beforeUid, list[oldIndex].Uid)
	if err != nil {
		return false
	}
	// Check and update the before_uid of the item before it in old order, aka the item with a before_uid pointing to this moved item
	if oldIndex > 0 {
		var oldbeforeUid int
		if oldIndex+1 < len(list) {
			oldbeforeUid = list[oldIndex+1].Uid
		} else {
			oldbeforeUid = 0
		}
		_, err = db.Exec("UPDATE queue SET before_uid = ? WHERE uid = ?", oldbeforeUid, list[oldIndex-1].Uid)
		if err != nil {
			return false
		}
	}
	// Update the before_uid of the item before in new order
	if newIndex > 0 {
		_, err = db.Exec("UPDATE queue SET before_uid = ? WHERE uid = ?", list[oldIndex].Uid, list[newIndex-1].Uid)
		if err != nil {
			return false
		}
	}
	return true
}
*/

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

	type userRow struct {
		user      *QueueUser
		beforeUid int // 0, -1, or >0
	}
	users := make(map[int]userRow)
	var uidList []int

	// 0. One(s) now handleing
	rows, err := db.Query("SELECT uid, nickname, level, gifts, topped FROM queue WHERE now = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
			continue
		}
		bid := 0
		if beforeUid.Valid {
			bid = int(beforeUid.Int64)
		}
		users[uid] = userRow{
			user: &QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        1,
			},
			beforeUid: bid,
		}
		uidList = append(uidList, uid)
	}

	// 1. Topped users
	rows, err = db.Query("SELECT uid, nickname, level, gifts, topped FROM queue WHERE now = 0 AND topped > 0 ORDER BY topped DESC, level DESC, gifts DESC, timestamp ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
			continue
		}
		bid := 0
		if beforeUid.Valid {
			bid = int(beforeUid.Int64)
		}
		users[uid] = userRow{
			user: &QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        0,
			},
			beforeUid: bid,
		}
		uidList = append(uidList, uid)
	}

	// 2. New & old abos
	rows, err = db.Query("SELECT uid, nickname, level, gifts, topped FROM queue WHERE now = 0 AND topped = 0 AND level > 0 ORDER BY level DESC, timestamp ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
			continue
		}
		bid := 0
		if beforeUid.Valid {
			bid = int(beforeUid.Int64)
		}
		users[uid] = userRow{
			user: &QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        0,
			},
			beforeUid: bid,
		}
		uidList = append(uidList, uid)
	}

	// 3. Cut-ins
	rows, err = db.Query("SELECT uid, nickname, level, gifts, topped FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts >= 52 ORDER BY gifts DESC, timestamp ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
			continue
		}
		bid := 0
		if beforeUid.Valid {
			bid = int(beforeUid.Int64)
		}
		users[uid] = userRow{
			user: &QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        0,
			},
			beforeUid: bid,
		}
		uidList = append(uidList, uid)
	}

	// 4. All users
	rows, err = db.Query("SELECT uid, nickname, level, gifts, topped FROM queue WHERE now = 0 AND topped = 0 AND level = 0 AND gifts < 52 AND timestamp > 0 ORDER BY timestamp ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, level, gifts int
		var nickname string
		var beforeUid sql.NullInt64
		if err := rows.Scan(&uid, &nickname, &level, &gifts, &beforeUid); err != nil {
			continue
		}
		bid := 0
		if beforeUid.Valid {
			bid = int(beforeUid.Int64)
		}
		users[uid] = userRow{
			user: &QueueUser{
				Uid:        uid,
				Uname:      nickname,
				GuardLevel: level,
				Gifts:      gifts,
				Now:        0,
			},
			beforeUid: bid,
		}
		uidList = append(uidList, uid)
	}

	/*
		ordered := make([]int, 0, len(users))
		placed := make(map[int]bool)

		// 1. Place users with beforeUid == 0
		for _, uid := range uidList {
			row := users[uid]
			if row.beforeUid == 0 {
				ordered = append(ordered, uid)
				placed[uid] = true
			}
		}

		// 2. Insert users with beforeUid > 0 before their target
		// Repeat until no more insertions can be made (to handle chains)
		inserted := true
		for inserted {
			inserted = false
			for uid, row := range users {
				if row.beforeUid > 0 && !placed[uid] {
					target := row.beforeUid
					for i, id := range ordered {
						if id == target {
							ordered = append(ordered[:i], append([]int{uid}, ordered[i:]...)...)
							placed[uid] = true
							inserted = true
							break
						}
					}
				}
			}
		}

		// 3. Append users with beforeUid == -1
		for _, uid := range uidList {
			row := users[uid]
			if row.beforeUid == -1 && !placed[uid] {
				ordered = append(ordered, uid)
				placed[uid] = true
			}
		}

		// 4. Append any users not yet placed (shouldn't happen, but just in case)
		for _, uid := range uidList {
			if !placed[uid] {
				ordered = append(ordered, uid)
			}
		}
	*/

	var result []*QueueUser
	/*
		for _, uid := range ordered {
			result = append(result, users[uid].user)
		}
	*/
	for _, uid := range uidList {
		result = append(result, users[uid].user)
	}
	return result, nil
}

func (q *Queue) UpdateGifts(u *QueueUser) bool {
	// 先判断是否需要递归调用 Add
	q.mu.RLock()
	needAdd := q.In(u)
	q.mu.RUnlock()
	if needAdd {
		// 递归调用放在锁外，避免死锁
		return q.Add(u)
	}

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
		q.mu.Unlock() // 先解锁
		ok := q.Add(&QueueUser{
			Uid:        u.Uid,
			Uname:      u.Uname,
			GuardLevel: u.GuardLevel,
			Gifts:      gft,
			Now:        0,
		})
		q.mu.Lock() // 重新加锁
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
