new Vue({el:'#add-content-images',template:`<div class="mb1">
<div v-for="(val, idx) in items" class="mb4">
  <fieldset class="p2 pb3">
    <legend>{{ idx + 1 }}</legend>
    <div class="mb2 flex flex-column">
      <label>URL</label>
      <input v-model="items[idx].URL" type="text" :name="'Images.' + idx + '.URL'">
    </div>
    <div class="mb2 flex flex-column">
      <label>{{ labels.caption }}</label>
      <input v-model="items[idx].Caption" type="text" :name="'Images.' + idx + '.Caption'">
    </div>
    <div class="mb3 flex flex-column">
      <label>{{ labels.linkto }}</label>
      <input v-model="items[idx].LinkTo" type="text" :name="'Images.' + idx + '.LinkTo'">
    </div>
    <a href="" class="btn-outline btn-blue py1 px2 rounded mt2" @click.prevent="remove(idx)">{{ labels.remove }}</a>
  </fieldset>
</div>
<a href="" class="btn btn-blue py1 px2 rounded" @click.prevent="add">{{ labels.more }}</a>
</div>`,created(){this.init()},data(){return{items:[],labels:{}}},methods:{init(){this.labels.caption=captionLabel||'Caption';this.labels.remove=removeLabel||'Remove';this.labels.more=moreLabel||'More';this.labels.linkto=linkToLabel||'Link';this.items=images||[]},add(){this.items.push({})},remove(idx){this.items.splice(idx,1)}}});
//# sourceMappingURL=add_images.js.map