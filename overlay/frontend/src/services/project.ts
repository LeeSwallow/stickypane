// Everything a project window asks for, in one value: the board, its
// notes, making notes and the settings. Board.svelte takes it as a prop,
// so a test and the dev pages hand it a fake.

import { boardService, type BoardApi } from './board'
import { createService, type CreateApi } from './create'
import { notesService, type NotesApi } from './notes'
import { settingsService, type SettingsApi } from './settings'

export type ProjectApi = BoardApi & NotesApi & CreateApi & SettingsApi

export const projectService: ProjectApi = { ...boardService, ...notesService, ...createService, ...settingsService }
