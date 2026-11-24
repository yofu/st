---
name: st-mcp-workflow
description: >-
  Fast, zero-exploration runbook for operating a live st.exe instance via the
  st MCP server. Use whenever the user asks to connect to ST, build or deform
  any structural model (frames, trusses, arches, domes, shells, slabs, walls),
  edit sections or supports, run static/buckling/vibration/nonlinear analyses,
  perform section rate checks (:srcal), display diagrams, or save INP/LST files.
---

# ST MCP Workflow Runbook

起動中の `st.exe` を `st` MCP サーバー経由で最短ターン・最小ツール呼び出しで操作するための手順書。
コマンドの網羅的な構文・オプション一覧は [references/commands.md](./references/commands.md) を参照する。

---

## 0. 高速実行の鉄則

1. **ソースコードを grep しない**: 通常のモデリング・解析・算定・表示・保存はこのスキルと [references/commands.md](./references/commands.md) のみで完結させる。
2. **`run_model_workflow` を主軸にする**: 可能な操作を 1 回のワークフローにまとめ、`wait_ms: 5000` と一意な `request_id` を指定する。
3. **接続 ID は初回のみでよい**: `connection_id` を一度渡せば同一 MCP セッション内で記憶されるが、明示して問題ない。
4. **レシートで検証する**: 完了時の `construction`, `first_changes`, `static.nodes`, `command_job.timing`, `gui.readback`, `gui.rated_elements`, `gui.srcan_rate` を確認し、不足や矛盾がない限り追加クエリ（`gui_status` / `query_*`）を呼ばない。

---

## 1. 環境・パス・単位系

### パス解決
- 未保存モデル（`gui.path == ""`）では相対パスが既定 `C:\Users\yofu8\st` に解決されて失敗する。
- ファイル指定（`:propsect`, `:read`, `:srcal`, `:bclng001`, `:vibeig`, `:save!`, `:writelst`）は **`$st/...`**（環境変数 `STDIR`、通常 `G:/マイドライブ/st`）または **フルパス**（スラッシュ `/` 区切り可）で渡す。
- よく使う参照モデルの例（必要に応じてユーザー指定パスを優先）:
  - `G:/マイドライブ/st/nakano/nakano02/nakano02_07.inp`（対応 LST: `.../nakano02_07.lst`）
- `static_analysis.result_path` は MCP サーバールート（`STDIR`）からの **相対パス**。既存ファイルへ上書き再解析する場合は `overwrite: true` を指定する（既定 `false` では `result_path already exists` エラー）。

### 単位系早見表（混同厳禁）

| 対象 | 長さ・断面寸法 | 荷重・反力・モーメント |
| :--- | :--- | :--- |
| MCP `construct`（`nodes`, `grid`, `panel_grid`） | **m** | — |
| MCP `construct.sections.dimensions_mm` | **mm** | — |
| MCP `construct.plate_sections` | — | 固定荷重 **`kgf/m2`**、`lload_tf_m2` **`tf/m2`**（3 要素） |
| MCP `edits.load` / 解析結果 `reaction` | 変位 **m** / 回転 **rad** | **`tf`** / **`tf·m`** |
| ST ネイティブコマンド（`:hkyou`, `:rpipe`, `:cpipe` 等） | **cm** | — |
| `.lst` の `CODE` 定義（`HKYOU`, `XFACE` 等） | **cm** | — |

---

## 2. `run_model_workflow` のステージ実行順

1 回の `run_model_workflow` は以下の**固定順**で実行される。`pre_commands` と `post_commands` を使うことで、「`:propsect` → グリッド・スラブ・曲面化 → 支点設定 → 解析 → `:srcal` → 検定比図表示」まで **1 回のワークフロー** で完結できる。

```text
 1. pre_commands       (construct 前のネイティブコマンド、例: :propsect / :read list)
 2. construct          (新規 PROP/SECT/NODE/ELEM/grid + update_sections(INP+LST同期) + plate_sections/panel_grid/plates + update_nodes)
 3. read_gui           (construct 無しで edits / editor_commands / static_analysis がある場合に自動同期)
 4. edit               (edits または editor_commands または proposal_id を適用)
 5. commands           (解析前のネイティブコマンド、例: :setboundary / :extractarclm / :bclng001 / :vibeig)
 6. static_analysis    (線形静的解析 L / X / Y、overwrite:true 対応、L は層境界未設定時に自動 1 層設定)
 7. show_result        (static_analysis 結果を GUI に自動ロード)
 8. post_commands      (static_analysis 後のネイティブコマンド、例: :srcal / 'srcanrate / 'srcancolor)
 9. display            (period / deformation / displacement_axes / stress / angles / focus)
10. save_copy          (ルート相対 .inp + .lst のコピー保存、GUI モデルは維持)
11. readback           (指定 period / node_ids の変位・固有値を読み戻し検証)
```

---

## 3. 汎用タスク別レシピ

### 3.1 接続確認
```json
// tool: gui_status
{ "connection_id": "<connection_id>" }
```

### 3.2 PROP / SECT のコピー + グリッド・面材・曲面化（1 回で完結）
`pre_commands` は `construct` より先に走るため、`:propsect` と `:read list` の直後に `construct`（`grid` + `panel_grid` + `update_nodes`）を 1 回のワークフローでまとめて実行できる。
```json
// tool: run_model_workflow
{
  "connection_id": "<id>",
  "pre_commands": [
    ":propsect G:/マイドライブ/st/nakano/nakano02/nakano02_07.inp",
    ":read list G:/マイドライブ/st/nakano/nakano02/nakano02_07.lst"
  ],
  "construct": {
    "grid": { "cells": [10, 10, 0], "pitch_m": [1, 1, 0], "section": 501, "type": "GIRDER" },
    "panel_grid": { "plane": "XY", "coordinate_m": 0, "expected_cells": 100, "section": 711, "type": "SLAB" }
  },
  "request_id": "propsect-grid-slabs",
  "wait_ms": 5000
}
```

### 3.3 モデル構築（任意形状：ラーメン・トラス・アーチ・ドーム・シェル・面材）

#### A. 軸平行グリッド + 面材（平面・立体ラーメン等）
- 直交線材は `construct.grid`（例：10×10 マス・1m ピッチ XY グリッドは `cells: [10, 10, 0]`, `pitch_m: [1, 1, 0]`）。
- 既存グリッドへの面材張りは `construct.panel_grid`（`plane: "XY"|"YZ"|"ZX"`, `coordinate_m`, `expected_cells`, `section`, `type: "SLAB"|"WALL"`）。
- `construct.grid` + `construct.panel_grid` + `construct.plate_sections` + `construct.update_nodes` は **1 回の `construct` に同梱可能**（`panel_grid` が平面上で生成された直後に `update_nodes` が 3D 曲面へ一括移動する）。

#### B. 任意 3D 形状（ドーム・アーチ・HP シェル・空間トラス・ラチス等）
- **原則（新規構築）**: 最初から 3D 座標を計算し、`construct.nodes` + `construct.elements` + `construct.plates` で 1 回に一括投入するか、`grid` + `panel_grid` + `update_nodes` で一括生成・変形する（最大 100,000 要素/節点）。
- **既存モデルを後から任意曲面へ変形する場合**:
  1. `gui_get_model` → `query_nodes(limit: 200)` で節点 ID・座標を取得。
  2. `run_model_workflow` の `construct.update_nodes: [{"id": ..., "xyz_m": [x, y, z]}, ...]` で最大 100,000 節点を **1 回の呼出で原子的に移動**する（非対称形状でも 100 件上限に当たらない）。
  3. 球面・放物面等で基準高さが未指定の場合は「四隅（または支持点）を Z=0 とする上向き凸」を既定とし、基準を回答に明記する。

### 3.4 既存断面の変更 + 再解析 + 再断面算定（1 回で完結）
`construct.update_sections` は既存線材断面（INP `SECT`）と鉄骨 LST（`CODE <id> S ...`）を **1 回のワークフローで同期更新**する（`sync_lst` は既定 `true`、`XFACE`/`YFACE` 等は保持）。さらに `static_analysis` と `post_commands` を同梱して再解析・再算定・検定比図表示まで **1 往復で完結**できる：
```json
{
  "connection_id": "<id>",
  "construct": {
    "update_sections": [
      { "id": 501, "shape": "HKYOU", "dimensions_mm": [200, 200, 8, 12] }
    ]
  },
  "static_analysis": {
    "period": "L",
    "result_path": "dome_r30_L.otl",
    "overwrite": true,
    "node_ids": [101, 161]
  },
  "post_commands": [
    ":srcal -skipshort -noreload G:/マイドライブ/st/dome_r30.rat",
    "'srcanrate long",
    "'srcancolor"
  ],
  "display": { "period": "L", "deformation": false, "displacement_axes": [] },
  "request_id": "sec501-h200-reanalyze-srcal",
  "wait_ms": 5000
}
```
※ 非鉄骨（RC/SRC）の LST を変更する場合や、旧 `stmcp` バイナリ使用時は `preview_lst_edits`（`proposal_id`）と `commands: [":hkyou 20 20 0.8 1.2 | :section 501 <-"]`（cm 単位）を併用する。

### 3.5 支点・材端結合・節点荷重
- **支点拘束 (`fixed`)・節点荷重 (`load`)**: `run_model_workflow.edits` で設定（100 件/回まで）。
  - 配列順: `[X, Y, Z, RX, RY, RZ]`
  - ピン支点: `[true, true, true, false, false, false]`
  - 剛支点: `[true, true, true, true, true, true]`
  - XY ローラー: `[false, false, true, false, false, false]`
- **材端ピン・剛結合**: `editor_commands` で `{ "element_ids": [...], "command": "BONDPIN" }` / `"BONDRIGID"` または `:bond pin` / `:bond rigid`。

### 3.6 線形静的解析（長期 `L` / 地震 `X`, `Y`）
- **新規モデルは層境界が未設定**なため、`commands` に `:setboundary <階数>`（ドーム・単層屋根・トラス等は `:setboundary 1`）を入れてから `static_analysis` を実行する（1 回のワークフローに同梱可能）。
- `static_analysis.node_ids` に代表節点（支点・頂点・スパン中央など）を入れておくと、レシートの `static.nodes` で変位と支点反力を即時確認できる。
```json
{
  "connection_id": "<id>",
  "edits": [
    { "node": 101, "fixed": [true, true, true, false, false, false] }
  ],
  "commands": [":setboundary 1"],
  "static_analysis": {
    "period": "L",
    "result_path": "model_L.otl",
    "node_ids": [101, 161]
  },
  "display": { "period": "L", "deformation": true, "displacement_axes": ["Z"] },
  "request_id": "pin-and-static-L",
  "wait_ms": 5000
}
```

### 3.7 断面算定（`:srcal`）と検定比図表示
- 事前に静的解析結果と LST がロード済みであること（未読込なら `:read list <path>.lst` を先頭に追加）。
- **長期のみ**: `-skipshort` を付ける。
- **メモリ上の編集済み LST を使う場合**: `-noreload` を付ける（付けないと `<otp>.lst` をディスクから読もうとする）。
- **出力パス**: 未保存モデルでも確実に書き出せるようフルパスまたは `$st/<name>.rat` を指定する。
- **検定比図表示**: `'srcanrate long`（短期含む場合は `'srcanrate`）＋ `'srcancolor`、さらに `display` で `deformation: false, displacement_axes: []` にすると見やすい。
```json
{
  "connection_id": "<id>",
  "commands": [
    ":srcal -skipshort -noreload G:/マイドライブ/st/model.rat",
    "'srcanrate long",
    "'srcancolor"
  ],
  "display": { "period": "L", "deformation": false, "displacement_axes": [] },
  "request_id": "srcal-long-display",
  "wait_ms": 5000
}
```
- **確認**: レシートの `gui.rated_elements`（算定対象本数）と `gui.srcan_rate`（長期 M+Q オンで `13`）で検証する。

### 3.8 座屈解析（`:bclng001`）と固有値振動解析（`:vibeig`）
- **必須前提**: 直前に必ず `:extractarclm` を実行する（未実行だと `frame isn't extracted to period L` エラー）。
- 座屈・振動ともに読み込まれたモードの `period` 名は **`B1`, `B2`, ...**（`B<次数>`）。
- 1 回の `run_model_workflow` で `:extractarclm` → 解析 → 読込 → 表示 → `readback` まで一括実行する：

```json
// 座屈解析（長期 L・一次モード）
{
  "connection_id": "<id>",
  "commands": [
    ":extractarclm",
    ":bclng001 -period=L -mode=1 G:/マイドライブ/st/model.bcl",
    ":read buckling G:/マイドライブ/st/model.bcl",
    ":color sect"
  ],
  "display": { "period": "B1", "deformation": true, "displacement_axes": ["Z"] },
  "readback": {
    "period": "B1",
    "node_ids": [101, 161],
    "require_deformation": true
  },
  "request_id": "buckling-L-mode1",
  "wait_ms": 5000
}
```
- **固有値振動解析**の場合は `:vibeig -period=L -mode=3 G:/マイドライブ/st/model.vib` → `:read vibrationeigenmode G:/マイドライブ/st/model.vib`（固有値 $\omega^2$ から固有周期 $T = 2\pi / \sqrt{\omega^2}$）。
- 固有値が `right` 上限（既定 `10.0`）より大きい場合でも自動探索されるが、高次モードや大固有値で必要なら `-right=100.0` 等を指定する。

### 3.9 保存（ダイアログ抑止）
- INP の保存（上書き確認ダイアログなし）: `:save! G:/マイドライブ/st/model.inp`
- LST の保存（メモリ上の最新 `CODE` 定義を書き出し）: `:writelst G:/マイドライブ/st/model.lst`
- 両方保存する場合は 1 回の `commands` にまとめられる（`:save!` の後に `:writelst` を並べる）。

---

## 4. 典型エラーと即座の対処

| エラーメッセージ | 原因 | 対処 |
| :--- | :--- | :--- |
| `open C:\Users\yofu8\st\...: cannot find the path` | 未保存モデルで相対パスを指定 | `$st/...` またはフルパスを指定（パス不明ならユーザーに確認） |
| `level isn't set up` | `Ai.Boundary` / 層境界が未設定 | `commands` に `:setboundary 1`（多層なら階数）を追加 |
| `frame isn't extracted to period L` | `:extractarclm` 未実行 | `:bclng001` / `:vibeig` / `:analysis` の直前に `:extractarclm` を追加 |
| `invalid SECT <id> or referenced PROP <p>` | 既存断面 ID に `construct.sections` を使用 | 既存断面変更は `preview_lst_edits` + `:hkyou ... \| :section <id> <-` を使用 |
| `result_path already exists` | `static_analysis.result_path` が既存ファイルと重複 | 新しい相対ファイル名に変更（例：`_v2_L.otl`） |
| `:srcal` で `.lst` が見つからない | 未保存モデルやリロード指定でディスクの `.lst` を探した | 事前に `:read list <path>.lst` し、`:srcal` に `-noreload` と明示出力パスを付与 |
