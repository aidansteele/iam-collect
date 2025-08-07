import { splitArnParts } from '@cloud-copilot/iam-utils'
import { AwsIamStore, OrganizationPolicyType, ResourceTypeParts } from '../AwsIamStore.js'
import { SqliteAdapter } from './SqliteAdapter.js'

export class SqliteAwsIamStore implements AwsIamStore {
  public sqliteAdapter: SqliteAdapter

  constructor(dbPath: string) {
    this.sqliteAdapter = new SqliteAdapter(dbPath)
  }

  private isEmptyContent(content: any): boolean {
    return (
      content === undefined ||
      content === null ||
      content === '' ||
      content === '{}' ||
      content === '[]' ||
      (Array.isArray(content) && content.length === 0) ||
      (typeof content === 'object' && Object.keys(content).length === 0)
    )
  }

  private normalizeContent(data: any): string {
    if (typeof data === 'string') {
      return data.trim()
    }
    return JSON.stringify(data, null, 2)
  }

  private parseContent<T>(content: string): T {
    try {
      return JSON.parse(content)
    } catch {
      return content as T
    }
  }

  async saveResourceMetadata(
    accountId: string,
    arn: string,
    metadataType: string,
    data: string | any
  ): Promise<void> {
    if (this.isEmptyContent(data)) {
      await this.deleteResourceMetadata(accountId, arn, metadataType)
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('resource_metadata', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase(),
      metadata_type: metadataType.toLowerCase(),
      data: content
    })
  }

  async listResourceMetadata(accountId: string, arn: string): Promise<string[]> {
    const results = this.sqliteAdapter.select('resource_metadata', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase()
    })
    return results.map((row: any) => row.metadata_type)
  }

  async getResourceMetadata<T, D extends T>(
    accountId: string,
    arn: string,
    metadataType: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('resource_metadata', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async deleteResourceMetadata(accountId: string, arn: string, metadataType: string): Promise<void> {
    this.sqliteAdapter.delete('resource_metadata', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })
  }

  async deleteResource(accountId: string, arn: string): Promise<void> {
    this.sqliteAdapter.delete('resource_metadata', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase()
    })
  }

  async listResources(accountId: string, options: ResourceTypeParts): Promise<string[]> {
    // Build dynamic query based on ResourceTypeParts
    let sql = `
      SELECT DISTINCT arn 
      FROM resource_metadata 
      WHERE account_id = ?
    `
    const params: any[] = [accountId.toLowerCase()]

    // Add ARN-based filtering using LIKE patterns
    // ARN format: arn:partition:service:region:account:resource
    const arnPatternParts: string[] = ['arn']
    
    if (options.partition) {
      arnPatternParts.push(options.partition)
    } else {
      arnPatternParts.push('%')
    }
    
    arnPatternParts.push(options.service)
    
    if (options.region) {
      arnPatternParts.push(options.region)
    } else {
      arnPatternParts.push('%')
    }
    
    if (options.account) {
      arnPatternParts.push(options.account)
    } else {
      arnPatternParts.push('%')
    }
    
    if (options.resourceType) {
      arnPatternParts.push(options.resourceType + '%')
    } else {
      arnPatternParts.push('%')
    }

    const arnPattern = arnPatternParts.join(':')
    sql += ` AND arn LIKE ?`
    params.push(arnPattern)

    // Debug logging
    console.log('SQL:', sql)
    console.log('Params:', params)
    
    // Debug: let's see what's actually in the database
    const allData = this.sqliteAdapter.select('resource_metadata', { account_id: accountId.toLowerCase() })
    console.log('All data for account:', allData)

    // Add metadata filtering if specified
    if (options.metadata) {
      for (const [key, value] of Object.entries(options.metadata)) {
        sql += ` AND arn IN (
          SELECT arn FROM resource_metadata 
          WHERE account_id = ? AND metadata_type = ? AND data LIKE ?
        )`
        params.push(accountId.toLowerCase(), key, `%${value}%`)
      }
    }

    const results = this.sqliteAdapter.exec(sql, params)
    console.log('Results:', results)
    return results.map((row: any) => row.arn)
  }

  async findResourceMetadata<T>(accountId: string, options: ResourceTypeParts): Promise<T[]> {
    const resources = await this.listResources(accountId, options)
    const results: T[] = []

    for (const arn of resources) {
      const metadataTypes = await this.listResourceMetadata(accountId, arn)
      for (const metadataType of metadataTypes) {
        const metadata = await this.getResourceMetadata(accountId, arn, metadataType)
        if (metadata) {
          results.push(metadata as T)
        }
      }
    }

    return results
  }

  async syncResourceList(
    accountId: string,
    options: ResourceTypeParts,
    desiredResources: string[]
  ): Promise<void> {
    const existingResources = await this.listResources(accountId, options)
    const resourcesToDelete = existingResources.filter(arn => !desiredResources.includes(arn))

    for (const arn of resourcesToDelete) {
      await this.deleteResource(accountId, arn)
    }
  }

  async deleteAccountMetadata(accountId: string, metadataType: string): Promise<void> {
    this.sqliteAdapter.delete('account_metadata', {
      account_id: accountId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })
  }

  async saveAccountMetadata(accountId: string, metadataType: string, data: any): Promise<void> {
    if (this.isEmptyContent(data)) {
      await this.deleteAccountMetadata(accountId, metadataType)
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('account_metadata', {
      account_id: accountId.toLowerCase(),
      metadata_type: metadataType.toLowerCase(),
      data: content
    })
  }

  async getAccountMetadata<T, D extends T>(
    accountId: string,
    metadataType: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('account_metadata', {
      account_id: accountId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async getOrganizationMetadata<T, D extends T>(
    organizationId: string,
    metadataType: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('organization_metadata', {
      organization_id: organizationId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async saveOrganizationMetadata(organizationId: string, metadataType: string, data: any): Promise<void> {
    if (this.isEmptyContent(data)) {
      await this.deleteOrganizationMetadata(organizationId, metadataType)
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('organization_metadata', {
      organization_id: organizationId.toLowerCase(),
      metadata_type: metadataType.toLowerCase(),
      data: content
    })
  }

  async deleteOrganizationMetadata(organizationId: string, metadataType: string): Promise<void> {
    this.sqliteAdapter.delete('organization_metadata', {
      organization_id: organizationId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })
  }

  async listOrganizationalUnits(organizationId: string): Promise<string[]> {
    const results = this.sqliteAdapter.select('organizational_unit_metadata', {
      organization_id: organizationId.toLowerCase()
    })
    const ouIds = new Set(results.map((row: any) => row.ou_id))
    return Array.from(ouIds)
  }

  async saveOrganizationalUnitMetadata(
    organizationId: string,
    ouId: string,
    metadataType: string,
    data: any
  ): Promise<void> {
    if (this.isEmptyContent(data)) {
      await this.deleteOrganizationalUnitMetadata(organizationId, ouId, metadataType)
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('organizational_unit_metadata', {
      organization_id: organizationId.toLowerCase(),
      ou_id: ouId.toLowerCase(),
      metadata_type: metadataType.toLowerCase(),
      data: content
    })
  }

  async deleteOrganizationalUnitMetadata(
    organizationId: string,
    ouId: string,
    metadataType: string
  ): Promise<void> {
    this.sqliteAdapter.delete('organizational_unit_metadata', {
      organization_id: organizationId.toLowerCase(),
      ou_id: ouId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })
  }

  async getOrganizationalUnitMetadata<T, D extends T>(
    organizationId: string,
    ouId: string,
    metadataType: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('organizational_unit_metadata', {
      organization_id: organizationId.toLowerCase(),
      ou_id: ouId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async deleteOrganizationalUnit(organizationId: string, ouId: string): Promise<void> {
    this.sqliteAdapter.delete('organizational_unit_metadata', {
      organization_id: organizationId.toLowerCase(),
      ou_id: ouId.toLowerCase()
    })
  }

  async deleteOrganizationPolicyMetadata(
    organizationId: string,
    policyType: OrganizationPolicyType,
    policyId: string,
    metadataType: string
  ): Promise<void> {
    this.sqliteAdapter.delete('organization_policy_metadata', {
      organization_id: organizationId.toLowerCase(),
      policy_type: policyType,
      policy_id: policyId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })
  }

  async saveOrganizationPolicyMetadata(
    organizationId: string,
    policyType: OrganizationPolicyType,
    policyId: string,
    metadataType: string,
    data: any
  ): Promise<void> {
    if (this.isEmptyContent(data)) {
      await this.deleteOrganizationPolicyMetadata(organizationId, policyType, policyId, metadataType)
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('organization_policy_metadata', {
      organization_id: organizationId.toLowerCase(),
      policy_type: policyType,
      policy_id: policyId.toLowerCase(),
      metadata_type: metadataType.toLowerCase(),
      data: content
    })
  }

  async getOrganizationPolicyMetadata<T, D extends T>(
    organizationId: string,
    policyType: OrganizationPolicyType,
    policyId: string,
    metadataType: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('organization_policy_metadata', {
      organization_id: organizationId.toLowerCase(),
      policy_type: policyType,
      policy_id: policyId.toLowerCase(),
      metadata_type: metadataType.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async deleteOrganizationPolicy(
    organizationId: string,
    policyType: OrganizationPolicyType,
    policyId: string
  ): Promise<void> {
    this.sqliteAdapter.delete('organization_policy_metadata', {
      organization_id: organizationId.toLowerCase(),
      policy_type: policyType,
      policy_id: policyId.toLowerCase()
    })
  }

  async listOrganizationPolicies(
    organizationId: string,
    policyType: OrganizationPolicyType
  ): Promise<string[]> {
    const results = this.sqliteAdapter.select('organization_policy_metadata', {
      organization_id: organizationId.toLowerCase(),
      policy_type: policyType
    })
    const policyIds = new Set(results.map((row: any) => row.policy_id))
    return Array.from(policyIds)
  }

  async syncRamResources(accountId: string, region: string | undefined, arns: string[]): Promise<void> {
    const normalizedRegion = region || 'global'
    
    // Delete existing resources not in the provided list
    const existingResults = this.sqliteAdapter.select('ram_resources', {
      account_id: accountId.toLowerCase(),
      region: normalizedRegion
    })
    
    const existingArns = existingResults.map((row: any) => row.arn)
    const arnsToDelete = existingArns.filter(arn => !arns.includes(arn))
    
    for (const arn of arnsToDelete) {
      this.sqliteAdapter.delete('ram_resources', {
        account_id: accountId.toLowerCase(),
        arn: arn.toLowerCase()
      })
    }
  }

  async saveRamResource(accountId: string, arn: string, data: any): Promise<void> {
    const arnParts = splitArnParts(arn)
    const region = arnParts.region || 'global'
    
    if (this.isEmptyContent(data)) {
      this.sqliteAdapter.delete('ram_resources', {
        account_id: accountId.toLowerCase(),
        arn: arn.toLowerCase()
      })
      return
    }

    const content = this.normalizeContent(data)
    this.sqliteAdapter.insertOrUpdate('ram_resources', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase(),
      region,
      data: content
    })
  }

  async getRamResource<T, D extends T>(
    accountId: string,
    arn: string,
    defaultValue?: D
  ): Promise<D extends undefined ? T | undefined : T> {
    const result = this.sqliteAdapter.selectOne('ram_resources', {
      account_id: accountId.toLowerCase(),
      arn: arn.toLowerCase()
    })

    if (!result) {
      return defaultValue as T
    }

    return this.parseContent<T>(result.data)
  }

  async listAccountIds(): Promise<string[]> {
    const results = this.sqliteAdapter.exec('SELECT DISTINCT account_id FROM resource_metadata UNION SELECT DISTINCT account_id FROM account_metadata UNION SELECT DISTINCT account_id FROM ram_resources')
    return results.map((row: any) => row.account_id)
  }

  async getIndex<T>(indexName: string, defaultValue: T): Promise<{ data: T; lockId: string }> {
    const result = this.sqliteAdapter.selectOne('indexes', {
      index_name: indexName.toLowerCase()
    })

    if (!result) {
      // Generate a new lock ID for new indexes
      const lockId = Math.random().toString(36).substr(2, 9)
      return { data: defaultValue, lockId }
    }

    return {
      data: this.parseContent<T>(result.data),
      lockId: result.lock_id
    }
  }

  async saveIndex<T>(indexName: string, data: T, lockId: string): Promise<boolean> {
    const existing = this.sqliteAdapter.selectOne('indexes', {
      index_name: indexName.toLowerCase()
    })

    // Check for optimistic locking conflict
    if (existing && existing.lock_id !== lockId) {
      return false
    }

    const content = this.normalizeContent(data)
    const newLockId = Math.random().toString(36).substr(2, 9)
    
    this.sqliteAdapter.insertOrUpdate('indexes', {
      index_name: indexName.toLowerCase(),
      data: content,
      lock_id: newLockId
    })

    return true
  }

  /**
   * Close the database connection
   */
  close(): void {
    this.sqliteAdapter.close()
  }
}