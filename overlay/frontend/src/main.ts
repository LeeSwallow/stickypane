import './fonts'
import './app.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { applyScale } from './lib/theme/scale'

// the size side of the design system (03 §3.11) as CSS variables
applyScale(document.documentElement)

mount(App, { target: document.getElementById('app')! })
