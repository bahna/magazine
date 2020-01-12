new Vue({
  el: '#choose-topic',
  
  template: `<select @change="goto" v-model="topic" class="my1 mr2 select">
    <option value='' :selected="topic === ''">{{ placeholder }}</option>
    <option v-for="o in options" :value="o.Slug">{{ o.Title }}</option>
  </select>`,
  
  created () {
    this.init()
  },
  
  computed: {
    placeholder () {
      let s = ''
      if (this.lang === 'en') {
        s = "All"
      } else if (this.lang === 'ru') {
        s = "Всё"
      } else if (this.lang === 'be') {
        s = "Усё"
      } 
      return s
    }
  },
  
  data () {
    return {
      lang: '',
      topic: '',
      options: []
    }
  },
  
  methods: {
    init () {
      this.options = topics; // values from the template
      let lang = window.location.pathname.split("/")[1]
      this.lang = lang
      let topic = window.location.pathname.split("/")[2]
      if (topic === 'login' || topic === 'signup') {
        topic = ''
      }
      this.topic = topic
    },
    
    goto (e) {
      let parts = window.location.pathname.split('/')
      let url
      
      if (e.target.value === '') {
        url = parts.slice(0,2).join('/') + '/'
      } else {
        parts[2] = e.target.value
        url = parts.slice(0,3).join('/') + '/'
        
      }
      
      window.location.pathname = url
    }
  }
})

new Vue({
  el: '#choose-language',
  
  template: `<select @change="goto" v-model="lang" class="my1 mr2 select">
  <option v-for="o in options" :value="o">{{ o }}</option>
  </select>`,
  
  created () {
    this.init()
  },
  
  data () {
    return {
      lang: '',
      options: []
    }
  },
  
  methods: {
    init () {
      this.options = languages; // values from the template
      
      let lang = window.location.pathname.split("/")[1]
      this.lang = lang
    },
    
    goto (e) {
      let parts = window.location.pathname.split('/')
      parts[1] = e.target.value
      //let url = parts.join('/')
      let url = "/" + parts[1] + "/"
      window.location.pathname = url
    }
  }
})
