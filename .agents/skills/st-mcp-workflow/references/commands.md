# ST Native Commands & MCP Quick Reference

`run_model_workflow.commands`（ライブ GUI で実行）および `run_model_workflow.editor_commands`（選択節点/要素付き編集）で使う頻出コマンドのリファレンス。

---

## 1. 断面形状コマンド（単位: **cm**）

すべて `| :section <sect_id> <-` にパイプして既存断面（`SECT` の `Figs[0]` と断面名）を更新、または `| :add sect <sect_id>` で新規断面を作成できる。

| コマンド | 引数（すべて cm） | 断面種別 |
| :--- | :--- | :--- |
| `:hkyou` | `h b tw tf` | H 形鋼（強軸） |
| `:hweak` | `h b tw tf` | H 形鋼（弱軸） |
| `:rpipe` | `h b tw tf` | 角形鋼管（BOX） |
| `:cpipe` | `d t` | 円形鋼管（外径・板厚） |
| `:tkyou` / `:tweak` | `h b tw tf` | T 形鋼（強軸/弱軸） |
| `:ckyou` / `:cweak` | `h b tw tf` | 溝形鋼（強軸/弱軸） |
| `:plate` | `h b` | 矩形プレート / 充実矩形 |
| `:angle` | `h b tw tf` | 山形鋼 |

例: `H-300x150x6.5x9` を断面 `502` に反映:
```text
:hkyou 30 15 0.65 0.9 | :section 502 <-
```
※ **断面算定用 `.lst` は別管理**。`preview_lst_edits` で同じ `CODE <sect_id>` を更新すること。

---

## 2. モデル編集コマンド（`editor_commands` / `commands`）

### 節点・幾何操作
- `:move x y z` — 選択節点 (`node_ids`) を `(x, y, z)` m 相対移動
- `:scale factor cx cy cz` — 中心 `(cx, cy, cz)` 基準でスケーリング（軸別は `:xscale` / `:yscale` / `:zscale factor coord`）
- `:mergenode {-eps=1e-4}` — 重複節点のマージ
- `:nodeduplication` / `:elemduplication` / `:nodenoreference` — 重複節点・重複要素・孤立節点の整理
- `:intersectall` — 交差線材の自動分割
- `:divide length l` — 線材の分割（`editor_commands`: `DIVIDEATMID`, `DIVIDEINN` も利用可）
- `:camber [axis] [add] period factor` — 解析変位に基づくキャンバー付与
- `:parabola` — 放物線形状調整

### 境界・結合・コードアングル・荷重
- `:conf [01_t]{6}` — 選択節点の拘束設定（例: ピン `111000`, 剛 `111111`, 自由 `000000`）
  - 簡易エディタ名: `CONFFIX`, `CONFPIN`, `CONFXYROLLER`, `CONFFREE`
- `:bond [pin,rigid,[0-9_t]{6}]` — 選択線材の材端結合（簡易エディタ名: `BONDPIN`, `BONDRIGID`）
- `:cang val` — 選択線材のコードアングル設定（deg）
- `:axis2cang n1 n2 [strong,weak]` — 2 節点方向へ強軸/弱軸を合わせる
- `:setcang [plate -enum=]` — 面法線等を基準にコードアングル設定
- `:pload position value` — 選択節点の荷重設定
- `:set {-sect=N} {-etype=TYPE}` — 選択要素の断面番号・部材種別変更（`COLUMN`, `GIRDER`, `BRACE`, `TRUSS`, `WALL`, `SLAB`）
- `:setboundary nfloor {-eps=0.001}` — 層境界（`Ai.Boundary`）を Z 座標から自動設定

---

## 3. ファイル入出力コマンド

| コマンド | 用途 |
| :--- | :--- |
| `:propsect <path>.inp` | 他モデルから `PROP` と `SECT` をコピー |
| `:insert <path>.inp <angle_deg>` | 選択節点位置に他モデルを挿入 |
| `:save! <path>.inp` | 確認ダイアログなしで INP 保存（カレントパス更新） |
| `:write! <path>.inp` | 確認ダイアログなしで INP 書き出し |
| `:writelst <path>.lst` | メモリ上の仮定断面リストを `.lst` 書き出し |
| `:read list <path>.lst` | `.lst` 読込 |
| `:read result <path>.otl` | 静的解析結果（`.otl`/`.ohx`/`.ohy`）読込 |
| `:read srcan <path>.rat` | 断面算定結果（`.rat`）読込 |
| `:read buckling <path>.bcl` | 座屈モード読込（`period = B1, B2, ...`） |
| `:read vibrationeigenmode <path>.vib` | 固有値振動モード読込（`period = B1, B2, ...`） |
| `:read zoubun <path>.<ext>` | 増分解析結果読込（`period = <EXT>@<lap>`） |

---

## 4. 解析・算定コマンド（`commands` で実行）

### 抽出（必須前処理）
- `:extractarclm` — `frame.Arclms["L"|"X"|"Y"]` を生成（`:analysis` / `:bclng001` / `:vibeig` / `:arclm101` の直前に必須）

### 静的・非線形解析
- 線形静的解析（推奨）: `run_model_workflow.static_analysis`（`period: "L"|"X"|"Y"`, 新規相対 `result_path`）
- ライブ静的/幾何非線形解析:
  `:analysis {-period=L|X|Y} {-all} {-nlgeom} {-nlmat} {-step=nlap;delta;start;max} {-arc -ds=...} {-wait} <path>`
- 弾塑性増分解析（プッシュオーバー）:
  `:arclm101 {-period=X|Y|L} {-nlgeom} {-step=nlap;delta} {-wait} <path>`

### 座屈・振動固有値解析
- 座屈解析:
  `:bclng001 {-period=L} {-mode=1} {-eps=1e-12} {-right=10.0} <path>.bcl`
- 固有値振動解析:
  `:vibeig {-period=L} {-mode=3} {-eps=1e-12} {-right=10.0} <path>.vib`

### 断面算定
- 許容応力度検定:
  `:srcal {-skipshort} {-noreload} {-verbose} {-fbold} {-torsion} {-qfact=2.0} {-wfact=2.0} {-bfact=1.0} <path>.rat`
  - 出力される主なファイル: `.rat`（検定比）, `.rlt`（一覧）, `.tst`（詳細）
- 鉄骨/RC 終局耐力・M-N 検討:
  `:steelyield ...` / `:rcyield ...` / `:nminteraction <sectcode>`

---

## 5. 表示制御コマンド（`commands` / `display`）

### 構造化 `display` パラメータ（優先使用）
```json
{
  "period": "L",
  "deformation": true,
  "deformation_scale": 100,
  "displacement_axes": ["Z"],
  "stress": [{ "type": "GIRDER", "component": "MZ" }],
  "angles": [30, -120],
  "focus": [5, 5, 0]
}
```
- `stress[].type`: `COLUMN`, `GIRDER`, `BRACE`, `TRUSS`, `WBRACE`, `SBRACE`, `WALL`, `SLAB`
- `stress[].component`: `N`, `NZ`, `QX`, `QY`, `MZ`, `MX`, `MY`
- `stress: []` / `displacement_axes: []` で非表示化

### Fig2 / Ex 表示コマンド
- `'srcanrate [long|short] [m|q]` — 検定比の数値表示オン（長期のみは `'srcanrate long`）
- `'srcancolor` / `:color rate` — 検定比カラー表示
- `:color [sect|n|rate|white|mono|strong]` — 部材色モード切替（通常表示へ戻すときは `:color sect`）
- `:view [top|front|back|right|left]` — 標準視点切替
- `'fit` — 全体フィット表示
- `'nodecode` / `'elemcode` / `'sectcode` / `'conf` / `'reaction` — 番号・支点・反力表示トグル
