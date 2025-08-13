<template>
  <div>
    <el-card style="margin: 5px">
      <!-- <el-alert
          title="温馨提示: 按住一行可以上下拖动排序~"
          type="success"
          show-icon
          style="margin-bottom: 10px">
      </el-alert> -->
      <el-button type="danger" icon="el-icon-delete" @click="removeAll">清除全部</el-button>
      <el-button type="primary" icon="el-icon-refresh" @click="syncData">同步</el-button>
      <el-divider direction="vertical"></el-divider>
      <el-button type="warning" icon="el-icon-video-pause" @click="pauseQueue">暂停排队</el-button>
      <el-button type="success" icon="el-icon-video-play" @click="continueQueue">继续排队</el-button>
    </el-card>
    <!-- <el-table-draggable> -->
      <el-table
          :data="tableData"
          border
          style="width: 100%; margin: 5px; border-radius: 5px;-webkit-box-shadow: 0 2px 12px 0 rgba(0,0,0,.1); box-shadow: 0 2px 12px 0 rgba(0,0,0,.1);">
        <el-table-column
            prop="nickname"
            label="序号"
            type="index"
            width="100">
        </el-table-column>
        <el-table-column
            prop="nickname"
            label="昵称">
        </el-table-column>
        <el-table-column
            prop="uid"
            label="UID">
        </el-table-column>
        <el-table-column
            prop="gifts"
            label="礼物金额">
        </el-table-column>
        <el-table-column label="航海等级">
          <template slot-scope="scope">
            <el-tag v-if="scope.row.level === '0'" type=""> 观众</el-tag>
            <el-tag v-else-if="scope.row.level === '1'" type="success">舰长</el-tag>
            <el-tag v-else-if="scope.row.level === '2'" type="warning">提督</el-tag>
            <el-tag v-else-if="scope.row.level === '3'" type="danger">总督</el-tag>
            <el-tag v-else-if="scope.row.level === '95'" type="success"><b>新</b>舰长</el-tag>
            <el-tag v-else-if="scope.row.level === '98'" type="warning"><b>新</b>提督</el-tag>
            <el-tag v-else-if="scope.row.level === '99'" type="danger"><b>新</b>总督</el-tag>
          </template>
        </el-table-column>
        <el-table-column fixed="right" label="操作">
          <template slot-scope="scope">
            <el-button v-if="scope.row.now == '1'" type="success" size="small" disabled>正在进行</el-button>
            <el-button v-else @click="start(scope.row)" type="success" size="small">开始</el-button>
            <el-button v-if="scope.row.now == '1'" type="warning" size="small" disabled>正在进行</el-button>
            <el-button v-else @click="top(scope.row)" type="warning" size="small">置顶</el-button>
            <el-button @click="removeUser(scope.row)" type="danger" size="small">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    <!-- </el-table-draggable> -->
  </div>
</template>

<script>
import client from '@/api/client'
//import ElTableDraggable from '@/components/Draggable/SortableElTable'

export default {
  name: 'Control',
  components: {
    //ElTableDraggable,
  },
  data() {
    return {
      tableData: this.$store.state.queue,
      testNow: 1
    }
  },
  methods: {
    removeUser(row) {
      client.emit('REMOVE_USER', row.uid)
    },
    top(row) {
      client.emit('TOP_USER', row.uid)
    },
    start(row) {
      client.emit('START_USER', row.uid)
      this.$message({
        message: `删除完成的，开始 ${row.nickname} 咯~`,
        duration: '1000',
        type: 'success'
      })
    },
    removeAll() {
      client.emit('REMOVE_ALL')
      this.$message({
        message: '清除全部排队',
        duration: '1000',
        type: 'success'
      })
    },
    syncData() {
      client.syncData()
      this.$message({
        message: '同步中~',
        duration: '1000',
        type: 'success'
      })
    },
    pauseQueue() {
      client.emit('PAUSE')
      this.$message({
        message: '已暂停',
        duration: '1000',
        type: 'warning'
      })
    },
    continueQueue() {
      client.emit('CONTINUE')
      this.$message({
        message: '已继续排队~',
        duration: '1000',
        type: 'success'
      })
    }
  },
  computed: {},
  mounted() {
    window.setInterval(() => {
      client.syncData()
    }, 1500)
  }
}
</script>
