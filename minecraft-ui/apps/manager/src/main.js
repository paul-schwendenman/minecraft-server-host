import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

import Home from './pages/Home.svelte'
import Maps from './pages/Maps.svelte'
import WorldDetail from './pages/WorldDetail.svelte'
import NotFound from './pages/NotFound.svelte'

const routes = {
  '/': Home,
  '/maps': Maps,
  '/maps/:world': WorldDetail,
  '*': NotFound
}

const app = mount(App, {
  target: document.getElementById('app'),
  props: { routes },
})

export default app
