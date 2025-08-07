import { beforeEach, describe, expect, it, afterEach } from 'vitest'
import { SqliteAwsIamStore } from './SqliteAwsIamStore.js'
import { SqliteAdapter } from './SqliteAdapter.js'
import * as fs from 'fs'
import * as path from 'path'

describe('SqliteAwsIamStore', () => {
  let store: SqliteAwsIamStore
  let tempDbPath: string

  beforeEach(() => {
    // Create a temporary database file for each test
    tempDbPath = path.join('/tmp', `test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}.db`)
    store = new SqliteAwsIamStore(tempDbPath)
  })

  afterEach(() => {
    // Clean up the database file
    store.close()
    if (fs.existsSync(tempDbPath)) {
      fs.unlinkSync(tempDbPath)
    }
  })

  describe('Resource Metadata', () => {
    const testAccountId = '123456789012'
    const testArn = 'arn:aws:iam::123456789012:role/test-role'
    const testMetadataType = 'trust-policy'
    const testData = { Version: '2012-10-17', Statement: [] }

    it('should save and retrieve resource metadata', async () => {
      await store.saveResourceMetadata(testAccountId, testArn, testMetadataType, testData)
      
      const result = await store.getResourceMetadata(testAccountId, testArn, testMetadataType)
      expect(result).toEqual(testData)
    })

    it('should return default value when metadata does not exist', async () => {
      const defaultValue = { default: true }
      const result = await store.getResourceMetadata(
        testAccountId,
        testArn,
        testMetadataType,
        defaultValue
      )
      expect(result).toEqual(defaultValue)
    })

    it('should list resource metadata types', async () => {
      await store.saveResourceMetadata(testAccountId, testArn, 'trust-policy', testData)
      await store.saveResourceMetadata(testAccountId, testArn, 'inline-policies', { policies: ['policy1'] })
      
      const metadataTypes = await store.listResourceMetadata(testAccountId, testArn)
      expect(metadataTypes).toContain('trust-policy')
      expect(metadataTypes).toContain('inline-policies')
      expect(metadataTypes).toHaveLength(2)
    })

    it('should delete specific resource metadata', async () => {
      await store.saveResourceMetadata(testAccountId, testArn, testMetadataType, testData)
      await store.saveResourceMetadata(testAccountId, testArn, 'another-type', { other: 'data' })
      
      await store.deleteResourceMetadata(testAccountId, testArn, testMetadataType)
      
      const result = await store.getResourceMetadata(testAccountId, testArn, testMetadataType)
      expect(result).toBeUndefined()
      
      const remainingTypes = await store.listResourceMetadata(testAccountId, testArn)
      expect(remainingTypes).toEqual(['another-type'])
    })

    it('should delete all resource metadata', async () => {
      await store.saveResourceMetadata(testAccountId, testArn, 'trust-policy', testData)
      await store.saveResourceMetadata(testAccountId, testArn, 'inline-policies', [])
      
      await store.deleteResource(testAccountId, testArn)
      
      const metadataTypes = await store.listResourceMetadata(testAccountId, testArn)
      expect(metadataTypes).toHaveLength(0)
    })

    it('should delete metadata when saving empty content', async () => {
      await store.saveResourceMetadata(testAccountId, testArn, testMetadataType, testData)
      
      // Save empty content should delete the metadata
      await store.saveResourceMetadata(testAccountId, testArn, testMetadataType, null)
      
      const result = await store.getResourceMetadata(testAccountId, testArn, testMetadataType)
      expect(result).toBeUndefined()
    })

    it('should handle string and object data types', async () => {
      const stringData = 'test string data'
      const objectData = { key: 'value', nested: { data: 123 } }
      
      await store.saveResourceMetadata(testAccountId, testArn, 'string-type', stringData)
      await store.saveResourceMetadata(testAccountId, testArn, 'object-type', objectData)
      
      const stringResult = await store.getResourceMetadata(testAccountId, testArn, 'string-type')
      const objectResult = await store.getResourceMetadata(testAccountId, testArn, 'object-type')
      
      expect(stringResult).toEqual(stringData)
      expect(objectResult).toEqual(objectData)
    })
  })

  describe('Account Metadata', () => {
    const testAccountId = '123456789012'
    const testMetadataType = 'metadata'
    const testData = { accountName: 'test-account', region: 'us-east-1' }

    it('should save and retrieve account metadata', async () => {
      await store.saveAccountMetadata(testAccountId, testMetadataType, testData)
      
      const result = await store.getAccountMetadata(testAccountId, testMetadataType)
      expect(result).toEqual(testData)
    })

    it('should return default value when account metadata does not exist', async () => {
      const defaultValue = { default: true }
      const result = await store.getAccountMetadata(testAccountId, testMetadataType, defaultValue)
      expect(result).toEqual(defaultValue)
    })

    it('should delete account metadata', async () => {
      await store.saveAccountMetadata(testAccountId, testMetadataType, testData)
      await store.deleteAccountMetadata(testAccountId, testMetadataType)
      
      const result = await store.getAccountMetadata(testAccountId, testMetadataType)
      expect(result).toBeUndefined()
    })

    it('should delete metadata when saving empty content', async () => {
      await store.saveAccountMetadata(testAccountId, testMetadataType, testData)
      await store.saveAccountMetadata(testAccountId, testMetadataType, {})
      
      const result = await store.getAccountMetadata(testAccountId, testMetadataType)
      expect(result).toBeUndefined()
    })
  })

  describe('Organization Metadata', () => {
    const testOrgId = 'o-test123456'
    const testMetadataType = 'metadata'
    const testData = { organizationName: 'test-org', features: ['SSO', 'SCPs'] }

    it('should save and retrieve organization metadata', async () => {
      await store.saveOrganizationMetadata(testOrgId, testMetadataType, testData)
      
      const result = await store.getOrganizationMetadata(testOrgId, testMetadataType)
      expect(result).toEqual(testData)
    })

    it('should return default value when organization metadata does not exist', async () => {
      const defaultValue = { default: true }
      const result = await store.getOrganizationMetadata(testOrgId, testMetadataType, defaultValue)
      expect(result).toEqual(defaultValue)
    })

    it('should delete organization metadata', async () => {
      await store.saveOrganizationMetadata(testOrgId, testMetadataType, testData)
      await store.deleteOrganizationMetadata(testOrgId, testMetadataType)
      
      const result = await store.getOrganizationMetadata(testOrgId, testMetadataType)
      expect(result).toBeUndefined()
    })
  })

  describe('Organizational Unit Metadata', () => {
    const testOrgId = 'o-test123456'
    const testOuId = 'ou-test123456'
    const testMetadataType = 'metadata'
    const testData = { ouName: 'test-ou', parentId: 'r-root123456' }

    it('should save and retrieve OU metadata', async () => {
      await store.saveOrganizationalUnitMetadata(testOrgId, testOuId, testMetadataType, testData)
      
      const result = await store.getOrganizationalUnitMetadata(testOrgId, testOuId, testMetadataType)
      expect(result).toEqual(testData)
    })

    it('should list organizational units', async () => {
      await store.saveOrganizationalUnitMetadata(testOrgId, 'ou-1', testMetadataType, testData)
      await store.saveOrganizationalUnitMetadata(testOrgId, 'ou-2', testMetadataType, testData)
      
      const ous = await store.listOrganizationalUnits(testOrgId)
      expect(ous).toContain('ou-1')
      expect(ous).toContain('ou-2')
      expect(ous).toHaveLength(2)
    })

    it('should delete OU metadata', async () => {
      await store.saveOrganizationalUnitMetadata(testOrgId, testOuId, testMetadataType, testData)
      await store.deleteOrganizationalUnitMetadata(testOrgId, testOuId, testMetadataType)
      
      const result = await store.getOrganizationalUnitMetadata(testOrgId, testOuId, testMetadataType)
      expect(result).toBeUndefined()
    })

    it('should delete entire OU', async () => {
      await store.saveOrganizationalUnitMetadata(testOrgId, testOuId, 'metadata', testData)
      await store.saveOrganizationalUnitMetadata(testOrgId, testOuId, 'policies', [])
      
      await store.deleteOrganizationalUnit(testOrgId, testOuId)
      
      const metaResult = await store.getOrganizationalUnitMetadata(testOrgId, testOuId, 'metadata')
      const policyResult = await store.getOrganizationalUnitMetadata(testOrgId, testOuId, 'policies')
      
      expect(metaResult).toBeUndefined()
      expect(policyResult).toBeUndefined()
    })
  })

  describe('Organization Policy Metadata', () => {
    const testOrgId = 'o-test123456'
    const testPolicyType = 'scps'
    const testPolicyId = 'p-test123456'
    const testMetadataType = 'metadata'
    const testData = { policyName: 'test-scp', content: { Version: '2012-10-17' } }

    it('should save and retrieve policy metadata', async () => {
      await store.saveOrganizationPolicyMetadata(
        testOrgId,
        testPolicyType,
        testPolicyId,
        testMetadataType,
        testData
      )
      
      const result = await store.getOrganizationPolicyMetadata(
        testOrgId,
        testPolicyType,
        testPolicyId,
        testMetadataType
      )
      expect(result).toEqual(testData)
    })

    it('should list organization policies', async () => {
      await store.saveOrganizationPolicyMetadata(testOrgId, testPolicyType, 'policy-1', testMetadataType, testData)
      await store.saveOrganizationPolicyMetadata(testOrgId, testPolicyType, 'policy-2', testMetadataType, testData)
      
      const policies = await store.listOrganizationPolicies(testOrgId, testPolicyType)
      expect(policies).toContain('policy-1')
      expect(policies).toContain('policy-2')
      expect(policies).toHaveLength(2)
    })

    it('should delete policy metadata', async () => {
      await store.saveOrganizationPolicyMetadata(
        testOrgId,
        testPolicyType,
        testPolicyId,
        testMetadataType,
        testData
      )
      
      await store.deleteOrganizationPolicyMetadata(
        testOrgId,
        testPolicyType,
        testPolicyId,
        testMetadataType
      )
      
      const result = await store.getOrganizationPolicyMetadata(
        testOrgId,
        testPolicyType,
        testPolicyId,
        testMetadataType
      )
      expect(result).toBeUndefined()
    })

    it('should delete entire policy', async () => {
      await store.saveOrganizationPolicyMetadata(testOrgId, testPolicyType, testPolicyId, 'metadata', testData)
      await store.saveOrganizationPolicyMetadata(testOrgId, testPolicyType, testPolicyId, 'targets', [])
      
      await store.deleteOrganizationPolicy(testOrgId, testPolicyType, testPolicyId)
      
      const metaResult = await store.getOrganizationPolicyMetadata(testOrgId, testPolicyType, testPolicyId, 'metadata')
      const targetsResult = await store.getOrganizationPolicyMetadata(testOrgId, testPolicyType, testPolicyId, 'targets')
      
      expect(metaResult).toBeUndefined()
      expect(targetsResult).toBeUndefined()
    })
  })

  describe('RAM Resources', () => {
    const testAccountId = '123456789012'
    const testArn = 'arn:aws:ram:us-east-1:123456789012:resource-share/test-share'
    const testData = { name: 'test-share', status: 'ACTIVE' }

    it('should save and retrieve RAM resource', async () => {
      await store.saveRamResource(testAccountId, testArn, testData)
      
      const result = await store.getRamResource(testAccountId, testArn)
      expect(result).toEqual(testData)
    })

    it('should return default value when RAM resource does not exist', async () => {
      const defaultValue = { default: true }
      const result = await store.getRamResource(testAccountId, testArn, defaultValue)
      expect(result).toEqual(defaultValue)
    })

    it('should sync RAM resources and delete non-matching ones', async () => {
      const arn1 = 'arn:aws:ram:us-east-1:123456789012:resource-share/share-1'
      const arn2 = 'arn:aws:ram:us-east-1:123456789012:resource-share/share-2'
      const arn3 = 'arn:aws:ram:us-east-1:123456789012:resource-share/share-3'
      
      // Save some initial resources
      await store.saveRamResource(testAccountId, arn1, { name: 'share-1' })
      await store.saveRamResource(testAccountId, arn2, { name: 'share-2' })
      await store.saveRamResource(testAccountId, arn3, { name: 'share-3' })
      
      // Sync with only arn1 and arn3
      await store.syncRamResources(testAccountId, 'us-east-1', [arn1, arn3])
      
      // arn2 should be deleted
      const result1 = await store.getRamResource(testAccountId, arn1)
      const result2 = await store.getRamResource(testAccountId, arn2)
      const result3 = await store.getRamResource(testAccountId, arn3)
      
      expect(result1).toEqual({ name: 'share-1' })
      expect(result2).toBeUndefined()
      expect(result3).toEqual({ name: 'share-3' })
    })
  })

  describe('Indexes', () => {
    const testIndexName = 'test-index'
    const testData = { version: 1, entries: ['item1', 'item2'] }

    it('should save and retrieve index with lock ID', async () => {
      const { lockId } = await store.getIndex(testIndexName, {})
      const success = await store.saveIndex(testIndexName, testData, lockId)
      
      expect(success).toBe(true)
      
      const { data } = await store.getIndex(testIndexName, {})
      expect(data).toEqual(testData)
    })

    it('should return default value for non-existent index', async () => {
      const defaultValue = { default: true }
      const { data } = await store.getIndex(testIndexName, defaultValue)
      
      expect(data).toEqual(defaultValue)
    })

    it('should fail to save with incorrect lock ID (optimistic locking)', async () => {
      const { lockId } = await store.getIndex(testIndexName, {})
      await store.saveIndex(testIndexName, testData, lockId)
      
      // Try to save with the old lock ID
      const success = await store.saveIndex(testIndexName, { new: 'data' }, lockId)
      expect(success).toBe(false)
    })

    it('should succeed with current lock ID', async () => {
      const { lockId: lockId1 } = await store.getIndex(testIndexName, {})
      await store.saveIndex(testIndexName, testData, lockId1)
      
      const { lockId: lockId2 } = await store.getIndex(testIndexName, {})
      const success = await store.saveIndex(testIndexName, { updated: 'data' }, lockId2)
      
      expect(success).toBe(true)
    })
  })

  describe('List Resources', () => {
    const testAccountId = '123456789012'

    beforeEach(async () => {
      // Set up test data
      await store.saveResourceMetadata(
        testAccountId,
        'arn:aws:iam::123456789012:role/test-role',
        'trust-policy',
        {}
      )
      await store.saveResourceMetadata(
        testAccountId,
        'arn:aws:iam::123456789012:user/test-user',
        'metadata',
        {}
      )
      await store.saveResourceMetadata(
        testAccountId,
        'arn:aws:s3:::test-bucket',
        'policy',
        {}
      )
    })

    it('should list resources by service', async () => {
      const iamResources = await store.listResources(testAccountId, { service: 'iam' })
      const s3Resources = await store.listResources(testAccountId, { service: 's3' })
      
      expect(iamResources).toHaveLength(2)
      expect(s3Resources).toHaveLength(1)
      expect(iamResources).toContain('arn:aws:iam::123456789012:role/test-role')
      expect(iamResources).toContain('arn:aws:iam::123456789012:user/test-user')
      expect(s3Resources).toContain('arn:aws:s3:::test-bucket')
    })

    it('should list resources by service and resource type', async () => {
      const roleResources = await store.listResources(testAccountId, {
        service: 'iam',
        resourceType: 'role'
      })
      const userResources = await store.listResources(testAccountId, {
        service: 'iam',
        resourceType: 'user'
      })
      
      expect(roleResources).toHaveLength(1)
      expect(userResources).toHaveLength(1)
      expect(roleResources).toContain('arn:aws:iam::123456789012:role/test-role')
      expect(userResources).toContain('arn:aws:iam::123456789012:user/test-user')
    })
  })

  describe('Sync Resource List', () => {
    const testAccountId = '123456789012'

    it('should remove resources not in desired list', async () => {
      const arn1 = 'arn:aws:iam::123456789012:role/role-1'
      const arn2 = 'arn:aws:iam::123456789012:role/role-2'
      const arn3 = 'arn:aws:iam::123456789012:role/role-3'
      
      // Save some initial resources
      await store.saveResourceMetadata(testAccountId, arn1, 'metadata', {})
      await store.saveResourceMetadata(testAccountId, arn2, 'metadata', {})
      await store.saveResourceMetadata(testAccountId, arn3, 'metadata', {})
      
      // Sync with only arn1 and arn3
      await store.syncResourceList(testAccountId, { service: 'iam', resourceType: 'role' }, [arn1, arn3])
      
      // Check that arn2 was removed
      const resources = await store.listResources(testAccountId, { service: 'iam', resourceType: 'role' })
      expect(resources).toContain(arn1)
      expect(resources).toContain(arn3)
      expect(resources).not.toContain(arn2)
    })
  })

  describe('List Account IDs', () => {
    it('should list all account IDs with data', async () => {
      await store.saveResourceMetadata('111111111111', 'arn:aws:iam::111111111111:role/role-1', 'metadata', {})
      await store.saveAccountMetadata('222222222222', 'metadata', {})
      await store.saveRamResource('333333333333', 'arn:aws:ram:us-east-1:333333333333:share/share-1', {})
      
      const accountIds = await store.listAccountIds()
      
      expect(accountIds).toContain('111111111111')
      expect(accountIds).toContain('222222222222')
      expect(accountIds).toContain('333333333333')
      expect(accountIds).toHaveLength(3)
    })
  })
})