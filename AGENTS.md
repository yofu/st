# Workspace Rules: ST MCP Operations

このリポジトリで起動中の `st.exe`（`st` MCP サーバー経由）の構造モデル作成・編集・解析・断面算定・描画を依頼された場合は、以下を厳守する。

## 1. スキル参照と探索の抑止
- `st` MCP を使ったモデル操作・解析を行う際は、まず [`.agents/skills/st-mcp-workflow/SKILL.md`](./.agents/skills/st-mcp-workflow/SKILL.md) を読むこと。
- **仕様確認のために `stlib/` や `cmd/stmcp/` を `Select-String` / `read_file` で探索しないこと**（明示的なコード修正依頼、または未知エラーでスキル記載の対処でも解決しない場合のみ例外）。
- `list_model_editors` はスキルおよびリファレンスに記載のないコマンドが必要な場合のみ使用する。

## 2. 速度最適化の基本原則
- 可能な限り `run_model_workflow` に複数操作をまとめ、`wait_ms: 5000`・一意な `request_id` を付けて実行する。
- 完了後はワークフローのレシート（`construction`, `first_changes`, `static.nodes`, `gui.readback`, `gui.display`, `gui.rated_elements`）から結果を回答し、重複する `gui_status` / `query_*` を呼ばない。
- モデルファイル直接編集や PowerShell スクリプトによる外部書き換えは行わず、ネイティブ MCP ツールのみを使う。

## 3. 最頻出の落とし穴（厳守）
- **単位系の違い**: `construct`（`nodes`, `grid`, `update_nodes` は **m**、`sections`/`update_sections.dimensions_mm` は **mm**）、ST ネイティブコマンド（`:hkyou` 等）と `.lst` は **cm**、`edits.load` は **tf / tf·m**。
- **未保存モデルのパス**: 未保存状態での相対パス指定は存在しない既定パスを参照する場合があるため、`:propsect` / `:read` / `:save!` / `:writelst` には `$st/...`（環境変数 `STDIR`）またはフルパスを使う。
- **既存断面の変更**: 新規断面は `construct.sections`、既存断面は `construct.update_sections`（INP と鉄骨 LST を 1 回で同期更新）を使う。`update_sections` 非対応の旧サーバーの場合は INP（`:hkyou ... | :section <ID> <-` 等）と LST（`preview_lst_edits`）の両方を更新する。
- **任意 3D 形状の構築・変形**: ドーム・アーチ・シェル・空間トラス等は、最初から 3D 座標の `construct.nodes` + `elements` + `plates` で一括生成するか、`construct.grid` + `panel_grid` + `update_nodes` を 1 回の `construct` にまとめて実行する。
- **ワークフロー一括バッチ**: `pre_commands`（`construct` 前、例 `:propsect` / `:read list`）と `post_commands`（`static_analysis` 後、例 `:srcal` / `'srcanrate` / `'srcancolor`）を活用し、可能な限り 1 回の `run_model_workflow` で完結させる。
- **新規モデル解析・座屈/振動解析**: 最新版では長期 `L` の層境界および未抽出時の `:extractarclm` は自動補完されるが、地震解析（`X`/`Y`）では `:setboundary <階数>` を明示し、モデル変更後の座屈・振動解析では `:extractarclm` を明示すると確実。
- **保存時のダイアログ抑止**: 上書き確認ダイアログを避けるため `:save! <path>.inp` を使い、仮定断面リスト変更時は `:writelst <path>.lst` も併せて保存する。
