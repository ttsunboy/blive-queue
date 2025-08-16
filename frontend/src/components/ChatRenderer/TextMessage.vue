<template>
  <!--    <img-shadow id="author-photo" height="24" width="24" class="style-scope yt-live-chat-text-message-renderer"-->
  <!--      :imgUrl="avatarUrl"-->
  <!--    ></img-shadow>-->
  <!--    <div id="content" class="style-scope yt-live-chat-text-message-renderer">-->
  <!--      <span id="timestamp" class="style-scope yt-live-chat-text-message-renderer">{{timeText}}</span>-->
  <!--      <yt-live-chat-author-chip class="style-scope yt-live-chat-text-message-renderer">-->
  <!--        <span id="author-name" dir="auto" class="style-scope yt-live-chat-author-chip" :type="authorTypeText">{{-->
  <!--          authorName-->
  <!--          }}&lt;!&ndash; 这里是已验证勋章 &ndash;&gt;-->
  <!--          <span id="chip-badges" class="style-scope yt-live-chat-author-chip"></span>-->
  <!--        </span>-->
  <!--        <span id="chat-badges" class="style-scope yt-live-chat-author-chip">-->
  <!--          <author-badge class="style-scope yt-live-chat-author-chip"-->
  <!--            :isAdmin="authorType === 2" :privilegeType="privilegeType"-->
  <!--          ></author-badge>-->
  <!--        </span>-->
  <!--      </yt-live-chat-author-chip>-->
  <!--      <span id="message" class="style-scope yt-live-chat-text-message-renderer">{{content}}</span>-->
  <tr>
    <td>
      <yt-live-chat-text-message-renderer class="queue-pos">
        <span v-if="queueNow == 1" id="message" class="style-scope yt-live-chat-text-message-renderer">NOW</span>
        <span v-else id="message" class="style-scope yt-live-chat-text-message-renderer" style="margin: 0 auto;">{{ queuePos }}</span>
      </yt-live-chat-text-message-renderer>
    </td>
    <td>
      <yt-live-chat-text-message-renderer class="queue-nickname" :author-type="authorTypeText">
        <span id="message" class="style-scope yt-live-chat-text-message-renderer" style="max-width: calc(100% - 120px);">{{ authorName }}</span>
      </yt-live-chat-text-message-renderer>
    </td>
    <td>
      <yt-live-chat-text-message-renderer class="queue-gifts">
        <span v-if="authorType == 0" id="message" class="style-scope yt-live-chat-text-message-renderer"  style="margin: 0 auto;">{{ gifts }}</span>
        <span v-else id="message" class="style-scope yt-live-chat-text-message-renderer" style="margin: 0 auto;">-</span>
      </yt-live-chat-text-message-renderer>
    </td>
  </tr>
  <!--    </div>-->
</template>

<script>
// import ImgShadow from './ImgShadow.vue'
// import AuthorBadge from './AuthorBadge.vue'
import * as constants from "./constants";
import * as utils from "@/utils";

// HSL
// const REPEATED_MARK_COLOR_START = [210, 100.0, 62.5]
// const REPEATED_MARK_COLOR_END = [360, 87.3, 69.2]

export default {
  name: "TextMessage",
  components: {
    // ImgShadow,
    // AuthorBadge
  },
  props: {
    avatarUrl: String,
    time: Date,
    authorName: String,
    authorType: Number,
    content: String,
    privilegeType: Number,
    repeated: Number,
    queuePos: Number,
    gifts: Number,
    queueNow: Number,
  },
  computed: {
    timeText() {
      return utils.getTimeTextHourMin(this.time);
    },
    authorTypeText() {
      // return 'member'
      switch (this.authorType) {
        case '95':
          return constants.AUTHOR_TYPE_TO_TEXT[1];
        case '98':
          return constants.AUTHOR_TYPE_TO_TEXT[2];
        case '99':
          return constants.AUTHOR_TYPE_TO_TEXT[3];
        default:
          return constants.AUTHOR_TYPE_TO_TEXT[this.authorType];
      }
    },
    // repeatedMarkColor() {
    //   let color
    //   if (this.repeated <= 2) {
    //     color = REPEATED_MARK_COLOR_START
    //   } else if (this.repeated >= 10) {
    //     color = REPEATED_MARK_COLOR_END
    //   } else {
    //     color = [0, 0, 0]
    //     let t = (this.repeated - 2) / (10 - 2)
    //     for (let i = 0; i < 3; i++) {
    //       color[i] = REPEATED_MARK_COLOR_START[i] + (REPEATED_MARK_COLOR_END[i] - REPEATED_MARK_COLOR_START[i]) * t
    //     }
    //   }
    //   return `hsl(${color[0]}, ${color[1]}%, ${color[2]}%)`
    // }
  },
};
</script>

<style>
yt-live-chat-text-message-renderer>#content>#message>.el-badge {
  margin-left: 10px;
}

yt-live-chat-text-message-renderer>#content>#message>.el-badge .el-badge__content {
  font-size: 12px !important;
  line-height: 18px !important;
  text-shadow: none !important;
  font-family: sans-serif !important;
  color: #fff !important;
  background-color: var(--repeated-mark-color) !important;
  border: none;
}
</style>

<style src="@/assets/css/youtube/yt-live-chat-text-message-renderer.css"></style>
<style src="@/assets/css/youtube/yt-live-chat-author-chip.css"></style>
