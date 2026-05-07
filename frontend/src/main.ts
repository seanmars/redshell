import '@fontsource-variable/geist';
import '@fontsource-variable/geist-mono';
import './assets/main.css';

import { createApp } from 'vue';

import App from './App.vue';
import { pinia } from './pinia';
import router from './router';

const app = createApp(App);

app.use(pinia);
app.use(router);

app.mount('#app');
