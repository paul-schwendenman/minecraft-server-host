import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

import Home from './pages/Home.svelte'
import Maps from './pages/Maps.svelte'
import NotFound from './pages/NotFound.svelte'

const routes = {
  '/': Home,
  '/maps': Maps,
  '*': NotFound
}

const app = mount(App, {
  target: document.getElementById('app'),
  props: { routes },
})

export default app
