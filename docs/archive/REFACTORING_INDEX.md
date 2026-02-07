# Flight3 ↔ SQLiter Refactoring Documentation

## Overview

This directory contains complete documentation for refactoring Flight3 to properly integrate with SQLiter, establishing a clear boundary of responsibilities.

**The Core Principle:**
```
Flight3: Scheme → DataSetPath (Resource Acquisition)
SQLiter: ColumnPath → Query (Data Querying)
```

---

## Document Index

### 📋 Start Here

1. **`ArchitectureSummary.md`** ⭐ **READ THIS FIRST**
   - Visual overview of the architecture
   - Data flow diagrams
   - Example scenarios
   - Quick reference

### 📖 Understanding the Architecture

2. **`ResponsibilityBoundary.md`** ⭐ **CRITICAL**
   - Precise boundary definition
   - Detailed responsibility matrix
   - Code examples for both sides
   - Communication protocol

3. **`RefactorSQLiter.md`**
   - Complete integration plan
   - Architecture rationale
   - Migration strategy
   - Benefits analysis

### 🔧 Implementation

4. **`ImplementationGuide.md`** ⭐ **FOLLOW THIS**
   - Step-by-step instructions
   - Code snippets
   - Verification steps
   - Troubleshooting guide
   - **Estimated time: 3 hours**

5. **`CleanUpTodo.md`**
   - Detailed HTML removal checklist
   - File-by-file changes
   - What to keep vs. remove
   - Verification checklist

### 📊 Analysis (Historical)

6. **`REFACTORING_ANALYSIS.md`**
   - Original redundancy analysis
   - Phase 1 completion notes
   - Historical context

7. **`REFACTORING_STATUS.md`**
   - Previous refactoring status
   - Blocking issues identified
   - Led to current solution

---

## Quick Start

**If you're ready to implement:**

1. Read `ArchitectureSummary.md` (5 min)
2. Read `ResponsibilityBoundary.md` (10 min)
3. Follow `ImplementationGuide.md` (3 hours)
4. Use `CleanUpTodo.md` as checklist

**Total time: ~3-4 hours**

---

## The Boundary (Quick Reference)

```
┌─────────────────────────────────────────────────────────────────┐
│                         BANQUET URL                              │
│  s3://user@host/data/sales.csv/tb0;name,amount;+date?limit=100  │
└─────────────────────────────────────────────────────────────────┘
         │                                    │
         ▼                                    ▼
┌──────────────────────────┐      ┌──────────────────────────────┐
│      FLIGHT3             │      │         SQLITER              │
│  Scheme → DataSetPath    │      │    ColumnPath → Query        │
└──────────────────────────┘      └──────────────────────────────┘
```

---

## Key Changes Summary

### What Gets Removed from Flight3

- ❌ `internal/flight/server.go` (entire file, 173 lines)
- ❌ HTML table rendering code
- ❌ Template initialization
- ❌ `sqliter.TableWriter` usage
- ❌ SQL query building (now uses `sqlite.Compose()`)

**Net: -172 lines**

### What Gets Added to Flight3

- ✅ SQLiter server initialization
- ✅ Mount `/_/data/` routes
- ✅ Redirect logic to SQLiter

**Net: +21 lines**

### What Stays in Flight3

- ✅ PocketBase integration
- ✅ Authentication handlers
- ✅ Rclone configuration
- ✅ File conversion (mksqlite)
- ✅ Cache management

### What Doesn't Change in SQLiter

- ✅ Everything! (already perfect)

---

## Architecture Before vs. After

### Before (Confused Responsibilities)

```
Flight3:
├── Auth ✅
├── Rclone ✅
├── Conversion ✅
├── HTML Rendering ❌ (shouldn't be here)
└── SQL Building ❌ (shouldn't be here)

SQLiter:
├── JSON API ✅
└── React UI ✅
```

### After (Clear Boundaries)

```
Flight3:
├── Auth ✅
├── Rclone ✅
├── Conversion ✅
└── Redirect to SQLiter ✅

SQLiter:
├── SQL Building ✅
├── Query Execution ✅
├── JSON API ✅
└── React UI ✅
```

---

## Benefits

### Code Quality
- ✅ 172 fewer lines in Flight3
- ✅ Single Responsibility Principle
- ✅ Clear separation of concerns
- ✅ Easier to test

### Maintainability
- ✅ Independent evolution
- ✅ Clear ownership
- ✅ Simpler debugging
- ✅ Better documentation

### User Experience
- ✅ Better UI (React + AG-Grid)
- ✅ Faster rendering
- ✅ More features
- ✅ Consistent experience

---

## Success Criteria

After implementation, verify:

- [ ] Flight3 has ZERO HTML rendering code
- [ ] All data queries go through SQLiter
- [ ] PocketBase admin UI still works
- [ ] Banquet URLs work correctly
- [ ] Local files work
- [ ] Remote files work
- [ ] Directory listings work
- [ ] Tests pass
- [ ] No errors in logs
- [ ] ~172 lines removed

---

## Timeline

| Phase | Duration |
|-------|----------|
| Reading documentation | 30 min |
| Implementation | 3 hours |
| Testing | 30 min |
| **Total** | **4 hours** |

---

## Filesystem Converter Note

**No conflicts!** ✅

- **mksqlite converter**: Converts local files → SQLite
- **flight3 IndexDirectory()**: Indexes remote directories → SQLite
- Both produce compatible `tb0` schema
- Different purposes, no duplication

---

## Questions?

Refer to the appropriate document:

- **"What's the boundary?"** → `ResponsibilityBoundary.md`
- **"How do I implement this?"** → `ImplementationGuide.md`
- **"What do I remove?"** → `CleanUpTodo.md`
- **"Why are we doing this?"** → `RefactorSQLiter.md`
- **"Show me examples"** → `ArchitectureSummary.md`

---

## Document Status

| Document | Status | Last Updated |
|----------|--------|--------------|
| ArchitectureSummary.md | ✅ Complete | 2026-01-30 |
| ResponsibilityBoundary.md | ✅ Complete | 2026-01-30 |
| RefactorSQLiter.md | ✅ Complete | 2026-01-30 |
| ImplementationGuide.md | ✅ Complete | 2026-01-30 |
| CleanUpTodo.md | ✅ Complete | 2026-01-30 |
| REFACTORING_ANALYSIS.md | 📚 Historical | 2026-01-29 |
| REFACTORING_STATUS.md | 📚 Historical | 2026-01-29 |

---

## Ready to Start?

1. **Read** `ArchitectureSummary.md`
2. **Understand** `ResponsibilityBoundary.md`
3. **Follow** `ImplementationGuide.md`
4. **Verify** with `CleanUpTodo.md`

Good luck! 🚀

---

## Contact

If you have questions or need clarification, refer back to these documents. They contain everything you need to successfully complete this refactoring.

**Remember the boundary:**
```
Flight3: Scheme → DataSetPath
SQLiter: ColumnPath → Query
```

Simple, clean, effective! 🎯
