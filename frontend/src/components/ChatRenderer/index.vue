<template>
  <yt-live-chat-renderer class="style-scope yt-live-chat-app" style="--scrollbar-width:11px;" hide-timestamps>
    <yt-live-chat-item-list-renderer class="style-scope yt-live-chat-renderer" allow-scroll>
      <div class="queue-sticky-header">
        <table>
          <colgroup>
            <col v-for="(w, i) in colWidths" :key="i" :style="{width: w + 'px'}">
          </colgroup>
          <thead>
            <tr>
              <!--
              <th v-if="waitingList.length <= 10" class="queue-pos">#</th>
              <th v-else class="queue-pos">共{{ waitingList.length }}人</th>
              -->
              <th class="queue-pos">
                <marquee direction="left" behavior="scroll" loop="-1" scrollamount="3" style="width: 3em; margin: 0 auto;">共{{ waitingList.length }}人</marquee>
              </th>
              <th class="queue-nickname">昵称</th>
              <th class="queue-gifts">礼物</th>
            </tr>
          </thead>
        </table>
      </div>
      <!-- 吸顶叫号行 -->
      <!--
      <div v-if="waitingList[0].now == '1'" class="queue-sticky-now">
        <table>
          <colgroup>
            <col v-for="(w, i) in colWidths" :key="i" :style="{width: w + 'px'}">
          </colgroup>
          <tbody>
            <template>
              <text-message
                :key="waitingList[0].uid"
                :authorName="waitingList[0].nickname"
                :authorType="waitingList[0].level"
                :queuePos="1"
                :gifts="waitingList[0].gifts"
                :queueNow="waitingList[0].now"
              />
            </template>
          </tbody>
        </table>
      </div>
      -->
      <!-- 主内容区 -->
      <div ref="scroller" id="item-scroller" class="style-scope yt-live-chat-item-list-renderer animated" style="margin-top: 24px;">
        <div ref="itemOffset" id="item-offset" class="style-scope yt-live-chat-item-list-renderer" style="height: 0px;">
          <div ref="items" id="items" class="style-scope yt-live-chat-item-list-renderer" style="overflow: hidden">
            <div id="content" class="style-scope yt-live-chat-text-message-renderer">
              <table>
                <tbody>
                  <template v-for="(w, pos) in waitingList">
                    <text-message
                      class="style-scope yt-live-chat-item-list-renderer"
                      :key="w.uid"
                      :authorName="w.nickname"
                      :authorType="w.level"
                      :queuePos="(parseInt(pos)+1-parseInt(waitingList[0].now))"
                      :gifts="w.gifts"
                      :queueNow="w.now"
                    ></text-message>
                  </template>
                </tbody>
              </table>
            </div>
          </div>
        </div>
        <div style="height: 24px;"></div>
      </div>
    </yt-live-chat-item-list-renderer>
  </yt-live-chat-renderer>
</template>

<script>
import TextMessage from './TextMessage.vue'

export default {
  name: 'ChatRenderer',
  components: {
    TextMessage,
  },
  data () {
    return {
      waitingList: this.$store.state.queue,
      stickyNowUser: null,
      showStickyHeader: false,
      headerHeight: 0,
      nowRowHeight: 0,
      colWidths: []
    }
  },
  computed: {},
  watch: {
    waitingList () {
      this.$nextTick(() => {
        this.updateStickyMetrics();
      });
    }
  },
  mounted () {
    this.$refs.scroller.addEventListener('scroll', this.handleScroll);
    this.$nextTick(() => {
      this.updateStickyMetrics();
      this.autoScroll();
    });
  },
  beforeDestroy () {
    this.$refs.scroller.removeEventListener('scroll', this.handleScroll);
  },
  methods: {
    updateStickyMetrics () {
      this.$nextTick(() => {
        // 表头高度
        const thead = this.$refs.items?.querySelector('thead');
        if (thead) this.headerHeight = thead.offsetHeight || 40;
        // 当前叫号行高度
        const nowRow = this.$refs.items?.querySelector('tr.queue-now');
        if (nowRow) this.nowRowHeight = nowRow.offsetHeight || 40;
        // 列宽
        const firstRow = this.$refs.items?.querySelector('tr');
        if (firstRow) {
          this.colWidths = Array.from(firstRow.children).map(td => td.offsetWidth);
        }
      });
    },
    handleScroll () {
      const scroller = this.$refs.scroller;
      if (!scroller) {
        this.stickyNowUser = null;
        this.showStickyHeader = false;
        return;
      }
      // 判断表头是否滚出
      const thead = this.$refs.items?.querySelector('thead');
      if (thead) {
        const theadRect = thead.getBoundingClientRect();
        const scrollerRect = scroller.getBoundingClientRect();
        this.showStickyHeader = theadRect.top < scrollerRect.top;
      } else {
        this.showStickyHeader = false;
      }

      // 判断当前叫号行是否滚出
      const nowRow = this.$refs.items?.querySelector('tr.queue-now');
      if (!nowRow) {
        this.stickyNowUser = null;
        return;
      }
      const rowRect = nowRow.getBoundingClientRect();
      const scrollerRect = scroller.getBoundingClientRect();
      if (rowRect.top < scrollerRect.top + (this.showStickyHeader ? this.headerHeight : 0)) {
        const nowUser = this.waitingList.find(u => u.now === 1);
        this.stickyNowUser = nowUser || null;
      } else {
        this.stickyNowUser = null;
      }
    },
    autoScroll () {
      if (this._scrollTimer) clearInterval(this._scrollTimer);

      const scroller = this.$refs.scroller;
      const items = this.$refs.items;
      const itemOffset = this.$refs.itemOffset;
      if (!scroller || !items || !itemOffset) return;

      // 只在有内容时同步高度
      if (items.clientHeight > 0) {
        itemOffset.style.height = `${items.clientHeight}px`;
      }

      let direction = 1;
      let pauseUntil = 0;
      const delay = 50;
      const step = 1;
      const topTarget = 0;

      this._scrollTimer = setInterval(() => {
        itemOffset.style.height = `${items.clientHeight}px`;
        const now = Date.now();
        const dynamicBottom = scroller.scrollHeight - scroller.clientHeight;

        if (now < pauseUntil) return;

        if (direction === 1 && scroller.scrollTop >= dynamicBottom - step) {
          direction = -1;
          pauseUntil = now + 1000;
          return;
        }
        if (direction === -1 && scroller.scrollTop <= topTarget + step) {
          direction = 1;
          pauseUntil = now + 8000;
          this.$emit('reach-top');
          return;
        }

        if (direction === 1) {
          scroller.scrollTop = Math.min(scroller.scrollTop + step, dynamicBottom);
        } else {
          scroller.scrollTop = Math.max(scroller.scrollTop - step, topTarget);
        }
      }, delay);

      // 暴露暂停方法
      this.pauseAutoScroll = (ms = 8000) => {
        pauseUntil = Date.now() + ms;
      }
    }
  }
}
</script>

<style src="@/assets/css/youtube/yt-html.css"></style>
<style src="@/assets/css/youtube/yt-live-chat-renderer.css"></style>
<style src="@/assets/css/youtube/yt-live-chat-item-list-renderer.css"></style>
