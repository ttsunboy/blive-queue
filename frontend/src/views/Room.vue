<template>
  <chat-renderer ref="renderer" @reach-top="onReachTop"></chat-renderer>
</template>

<script>
import client from '@/api/client'
import * as chatConfig from '@/api/chatConfig'
import ChatRenderer from '@/components/ChatRenderer'

export default {
  name: 'Room',
  components: {
    ChatRenderer
  },
  props: {
    roomId: {
      type: Number,
      default: null
    },
    strConfig: {
      type: Object,
      default: () => ({})
    }
  },
  data () {
    return {
      config: { ...chatConfig.DEFAULT_CONFIG },
    }
  },
  computed: {},
  mounted () {
    // 提示用户已加载
    this.$message({
      message: 'Loaded',
      duration: '500'
    })
    this.$refs.renderer.autoScroll()
  },
  methods: {
    onReachTop () {
      client.syncData();
      // 保证同步后停留8秒
      if (this.$refs.renderer && this.$refs.renderer.pauseAutoScroll) {
        this.$refs.renderer.pauseAutoScroll(8000);
      }
    }
  }
}
</script>
