import { render } from '@testing-library/svelte'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { writable } from 'svelte/store'
import '@testing-library/jest-dom'
import * as stores from '../../src/stores.js'
import ServerDetails from '../../src/components/ServerDetails.svelte'

describe('ServerDetails', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  describe('server details returned successfully', () => {
    it('handles resolved promise', async () => {
      const fakeDetails = writable({
        players: { max: 20, online: 0 }
      })
      vi.spyOn(stores, 'details', 'get').mockReturnValue(fakeDetails)

      const { getByText } = render(ServerDetails)
      expect(getByText('The server has no active players.')).toBeInTheDocument()
    })

    it('handles no active users', () => {
      vi.spyOn(stores, 'details', 'get').mockReturnValue(
        writable({
          players: { max: 20, online: 0 }
        })
      )
      const { getByText } = render(ServerDetails)
      expect(getByText('The server has no active players.')).toBeInTheDocument()
    })

    it('handles 1 active user', () => {
      vi.spyOn(stores, 'details', 'get').mockReturnValue(
        writable({
          players: {
            max: 20,
            online: 1,
            sample: [{ id: 'cdce37cd', name: 'example' }]
          }
        })
      )
      const { getByText, getAllByRole } = render(ServerDetails)
      expect(getByText('The server has 1 active player:')).toBeInTheDocument()
      expect(getAllByRole('listitem')).toHaveLength(1)
      expect(getByText('example')).toBeInTheDocument()
    })

    it('handles multiple active users', () => {
      vi.spyOn(stores, 'details', 'get').mockReturnValue(
        writable({
          players: {
            max: 20,
            online: 2,
            sample: [
              { id: '1', name: 'example' },
              { id: '2', name: 'example2' }
            ]
          }
        })
      )
      const { getByText, getAllByRole } = render(ServerDetails)
      expect(getByText('The server has 2 active players:')).toBeInTheDocument()
      expect(getAllByRole('listitem')).toHaveLength(2)
      expect(getByText('example')).toBeInTheDocument()
      expect(getByText('example2')).toBeInTheDocument()
    })
  })

  describe('server details are still loading', () => {
    it('displays loading message', () => {
      const pending = new Promise(() => { })
      vi.spyOn(stores, 'details', 'get').mockReturnValue(writable(pending))

      const { getByText } = render(ServerDetails)
      expect(getByText('Loading details...')).toBeInTheDocument()
    })
  })

  describe('server details failed to load', () => {
    it('displays an error message', async () => {
      const failing = writable(Promise.reject('Error'))
      vi.spyOn(stores, 'details', 'get').mockReturnValue(failing)

      const { findByText } = render(ServerDetails)
      const errorEl = await findByText('Failed to load details.')
      expect(errorEl).toBeInTheDocument()
      expect(errorEl.className).toContain('text-red-700')
    })

  })
})
