// Everything a project window asks for, in one value: the board, its
// notes, making notes, the settings and where notes sit (the canvas).
// Board.svelte takes it as a prop, so a test and the dev pages hand it a
// fake.

import { boardService, type BoardApi } from './board'
import { canvasService, type CanvasApi } from './canvas'
import { createService, type CreateApi } from './create'
import { notesService, type NotesApi } from './notes'
import { settingsService, type SettingsApi } from './settings'

export type ProjectApi = BoardApi & NotesApi & CreateApi & SettingsApi & CanvasApi

export const projectService: ProjectApi = { ...boardService, ...notesService, ...createService, ...settingsService, ...canvasService }
