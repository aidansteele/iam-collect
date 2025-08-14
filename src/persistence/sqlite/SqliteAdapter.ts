const Database = require('better-sqlite3')

export interface SqliteTable {
  name: string
  schema: string
}

/**
 * SQLite adapter for AWS IAM Store operations.
 * Handles database initialization and provides low-level database operations.
 */
export class SqliteAdapter {
  private db: any

  constructor(dbPath: string) {
    this.db = new Database(dbPath)
    this.initializeTables()
  }

  private initializeTables(): void {
    // Enable foreign keys
    this.db.pragma('foreign_keys = ON')

    // Create resource metadata table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS resource_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        account_id TEXT NOT NULL,
        arn TEXT NOT NULL,
        metadata_type TEXT NOT NULL,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(account_id, arn, metadata_type)
      )
    `)

    // Create account metadata table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS account_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        account_id TEXT NOT NULL,
        metadata_type TEXT NOT NULL,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(account_id, metadata_type)
      )
    `)

    // Create organization metadata table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS organization_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        organization_id TEXT NOT NULL,
        metadata_type TEXT NOT NULL,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(organization_id, metadata_type)
      )
    `)

    // Create organizational unit metadata table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS organizational_unit_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        organization_id TEXT NOT NULL,
        ou_id TEXT NOT NULL,
        metadata_type TEXT NOT NULL,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(organization_id, ou_id, metadata_type)
      )
    `)

    // Create organization policy metadata table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS organization_policy_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        organization_id TEXT NOT NULL,
        policy_type TEXT NOT NULL,
        policy_id TEXT NOT NULL,
        metadata_type TEXT NOT NULL,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(organization_id, policy_type, policy_id, metadata_type)
      )
    `)

    // Create RAM resources table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS ram_resources (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        account_id TEXT NOT NULL,
        arn TEXT NOT NULL,
        region TEXT,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(account_id, arn)
      )
    `)

    // Create indexes table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS indexes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        index_name TEXT NOT NULL UNIQUE,
        data TEXT NOT NULL,
        lock_id TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `)

    // Create indexes for better performance
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_resource_metadata_account_arn ON resource_metadata(account_id, arn)'
    )
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_account_metadata_account ON account_metadata(account_id)'
    )
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_organization_metadata_org ON organization_metadata(organization_id)'
    )
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_organizational_unit_metadata_org_ou ON organizational_unit_metadata(organization_id, ou_id)'
    )
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_organization_policy_metadata_org_type ON organization_policy_metadata(organization_id, policy_type)'
    )
    this.db.exec(
      'CREATE INDEX IF NOT EXISTS idx_ram_resources_account_region ON ram_resources(account_id, region)'
    )
  }

  /**
   * Insert or update a record with conflict resolution
   */
  insertOrUpdate(table: string, data: Record<string, any>): void {
    const keys = Object.keys(data)
    const placeholders = keys.map(() => '?').join(', ')
    const updateClause = keys
      .filter((k) => k !== 'created_at')
      .map((k) => `${k} = excluded.${k}`)
      .join(', ')

    const sql = `
      INSERT INTO ${table} (${keys.join(', ')}) 
      VALUES (${placeholders})
      ON CONFLICT DO UPDATE SET ${updateClause}, updated_at = CURRENT_TIMESTAMP
    `

    const stmt = this.db.prepare(sql)
    stmt.run(...Object.values(data))
  }

  /**
   * Select records from a table with optional where conditions
   */
  select(table: string, where?: Record<string, any>): any[] {
    let sql = `SELECT * FROM ${table}`
    let params: any[] = []

    if (where && Object.keys(where).length > 0) {
      const conditions = Object.keys(where)
        .map((k) => `${k} = ?`)
        .join(' AND ')
      sql += ` WHERE ${conditions}`
      params = Object.values(where)
    }

    const stmt = this.db.prepare(sql)
    return stmt.all(...params)
  }

  /**
   * Select a single record from a table
   */
  selectOne(table: string, where: Record<string, any>): any {
    const conditions = Object.keys(where)
      .map((k) => `${k} = ?`)
      .join(' AND ')
    const sql = `SELECT * FROM ${table} WHERE ${conditions} LIMIT 1`
    const stmt = this.db.prepare(sql)
    return stmt.get(...Object.values(where))
  }

  /**
   * Delete records from a table
   */
  delete(table: string, where: Record<string, any>): void {
    const conditions = Object.keys(where)
      .map((k) => `${k} = ?`)
      .join(' AND ')
    const sql = `DELETE FROM ${table} WHERE ${conditions}`
    const stmt = this.db.prepare(sql)
    stmt.run(...Object.values(where))
  }

  /**
   * Execute a custom SQL query
   */
  exec(sql: string, params: any[] = []): any {
    const stmt = this.db.prepare(sql)
    return stmt.all(...params)
  }

  /**
   * Close the database connection
   */
  close(): void {
    this.db.close()
  }

  /**
   * Get the underlying database instance for advanced operations
   */
  getDatabase(): any {
    return this.db
  }
}
