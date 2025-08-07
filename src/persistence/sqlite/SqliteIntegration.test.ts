import { beforeEach, describe, expect, it, afterEach } from 'vitest'
import { createStorageClient } from '../util.js'
import { SqliteAwsIamStore } from './SqliteAwsIamStore.js'
import * as fs from 'fs'
import * as path from 'path'

describe('SQLite Integration Test', () => {
  let tempDbPath: string

  beforeEach(() => {
    tempDbPath = path.join('/tmp', `integration-test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}.db`)
  })

  afterEach(() => {
    if (fs.existsSync(tempDbPath)) {
      fs.unlinkSync(tempDbPath)
    }
  })

  it('should create SqliteAwsIamStore via createStorageClient', () => {
    const storageConfig = {
      type: 'sqlite' as const,
      dbPath: tempDbPath
    }

    const store = createStorageClient(storageConfig, 'aws')
    
    expect(store).toBeInstanceOf(SqliteAwsIamStore)
  })

  it('should work end-to-end with createStorageClient', async () => {
    const storageConfig = {
      type: 'sqlite' as const,
      dbPath: tempDbPath
    }

    const store = createStorageClient(storageConfig, 'aws')
    
    // Test basic operations
    const testAccountId = '123456789012'
    const testArn = 'arn:aws:iam::123456789012:role/test-role'
    const testData = { Version: '2012-10-17', Statement: [] }

    await store.saveResourceMetadata(testAccountId, testArn, 'trust-policy', testData)
    const result = await store.getResourceMetadata(testAccountId, testArn, 'trust-policy')
    
    expect(result).toEqual(testData)

    // Test cleanup
    if (store instanceof SqliteAwsIamStore) {
      store.close()
    }
  })
})