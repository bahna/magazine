new Vue({
  el: '#content-type-select',

  template: `<div class="mb4">
    <div class="mb2 flex flex-column">
      <label>{{ this.labels.type }}</label>
      <select name="Type" v-model="contentType" required>
        <option v-for="(k, v) in this.contentTypes" :value="v">{{ k }}</option>
      </select>
    </div>
    
    <div v-if="contentType == 5">
      <div class="mb2 flex flex-column">
        <label>{{ this.labels.eventStartTime }}</label>
        <input type="datetime-local" name="EventStart" placeholder="2018-05-01T24:00" :value="contentEventStart">
      </div>
      <div class="mb2 flex flex-column">
        <label>{{ this.labels.eventLocation }}</label>
        <input type="text" name="Location" :value="contentLocation">
      </div>
    </div>

    <div v-if="contentType == 1">
      <div class="mb2 flex flex-column">
        <label>{{ this.labels.linkTo }}</label>
        <input type="text" name="LinkTo" placeholder="https://bahna.land/" :value="contentLinkTo">
      </div>
    </div>
  </div>`,

  created () {
    this.init()
  },

  data () {
    return {
      contentTypes: {},
      contentType: 0,
      labels: {},
      contentLinkTo: null,
      contentEventStart: null,
      contentType: null
    }
  },

  methods: {
    init () {
      this.labels.type = chooseTypeLabel || 'Choose Type'
      this.labels.eventStartTime = eventStartTimeLabel || 'Event Start Time'
      this.labels.linkTo = linkToLabel || 'Link To'
      this.labels.eventLocation = eventLocationLabel || 'Location'
      this.contentTypes = contentTypes
      this.contentLinkTo = contentLinkTo 
      this.contentEventStart = contentEventStart
      this.contentType = contentType
      this.contentLocation = contentLocation
    }
  }
})
