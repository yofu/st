# 図面設定と整合性検査（第①段階）

解析モデルと図面の調整情報を分離するため、同名の
`<モデル名>.drawing.json` を使用します。INP・LST・KJNを書き換えず、
再出力でも符号・注記・確認済みの高さ・補助通り・文字位置を引き継ぎます。

## GUIのexコマンド

```text
:drawinginit
:drawingcheck
:drawings sheets.dxf -plans=3 -grids=Y2,X1 -sections
```

- `:drawinginit [設定ファイル]`: 現在のモデルとKJNから初期設定を作成します。
  既存ファイルは上書きしません。GLは、明示されたGL通りがなければ未確定です。
  推定した階・通りには `source: "inferred"`、`confirmed: false` を付けます。
- `:drawingcheck [設定ファイル]`: エラー・警告を節点接続や断面番号とともに履歴へ出力します。
  ファイルを省略すると同名の設定を読みます。未作成なら初期候補を検査します。
- `:drawings` / `:dxfsections`: 同名の設定を自動で読みます。
  別の設定は `-settings=other.drawing.json` で指定できます。
- `:plan ... -dim`、`:jiku ... -dim`、`:crosssection ... -dim` も
  同名設定を使用します。従来の非寸法出力は設定の自動読み込み対象外です。

## 単位と指定の優先順位

- GL・FL高さ、通り位置、部材の寄り、天端差、注記の座標: **mm**。
  高さ・寄りはモデルのグローバルXYZ座標に対応します。
- 符号・注記の移動量: **用紙上のmm**。縮尺が変わっても用紙上の調整量を維持します。
- 部材設定は `sections[断面番号]` → `members[部材番号]` の順に適用します。
  省略した項目は継承し、`0`、空の注記配列は明示的な上書きです。
- `levels` は図面で使う階の一覧です。`floor` は `-plans` と同じ階番号で、
  階選択用のAI境界値とは独立したFL高さを設定します。未割当ては `floor: 0`。
- `grids` は同名KJN通りを図面上で上書きし、新しい名前なら補助通りを追加します。
  `axis: "X"` はX座標一定、`"Y"` はY座標一定の通りです。
- 元データと設定が矛盾する場合、推測で解析モデルを修正しません。

例（実際の断面・部材番号と階に置き換えて使用）:

```json
{
  "version": 1,
  "ground_level_mm": 0,
  "levels": [
    {"name": "2FL", "floor": 2, "elevation_mm": 2850, "source": "reviewed", "confirmed": true},
    {"name": "3FL", "floor": 3, "elevation_mm": 5650, "source": "reviewed", "confirmed": true}
  ],
  "grids": [
    {"name": "X2", "axis": "X", "position_mm": 815, "source": "reviewed", "confirmed": true}
  ],
  "sections": {
    "501": {"symbol": "G51"}
  },
  "members": {
    "1234": {
      "symbol": "G51A",
      "level": "3FL",
      "offset_mm": [0, 20, 0],
      "top_offset_mm": -70,
      "label_offset_paper_mm": [2, 1],
      "notes": ["個別の確認済み注記"],
      "expected": {"section": 501, "nodes": [101, 102]}
    }
  },
  "annotations": [
    {
      "id": "plan3-note",
      "view": "plan:3",
      "text": "確認済みの共通注記",
      "position_mm": [500, -1000],
      "offset_paper_mm": [0, 0]
    }
  ]
}
```

`expected` は設定対象の断面と接続節点を記録します。節点順の反転は許容します。
同じ番号が別の部材になった場合や削除された場合は出力を止めます。
部材が移動した場合はモデル改訂の警告を出し、部材に追従する注記は移動後の位置を使用します。
図面独自の符号・注記・文字位置が違う部材は、連続部材統合で設定が失われないよう分けます。

注記の表示対象 `view` は `all`、`plan:3`、`elevation:Y2` などです。
`position_mm` と `element`（部材中央への追従）のどちらか一方を指定します。
改行は `\n` を使えます。IDは設定内で一意にします。
指定した符号位置は自動衝突回避で再移動しません。
現段階では注記同士の衝突回避は行わず、後続の⑤で改善します。

**①での寄り・天端差は保存・継承・検査までです。**
`offset_mm` と `top_offset_mm` の実部材外形への反映は③で実装します。
これらがある場合は `geometry_metadata_only` をレポートに出します。
FL・通り・符号・注記・符号位置は現在のDXFに反映します。

## MCP

1. `query_dxf_settings`: 既存設定、または自動生成候補と検査結果を取得。
2. `preview_dxf_settings`: 完全な `settings` を渡し、変更前後・継承後の値を確認。
   新しい部材指定の `expected` は自動で補います。既存の不一致は勝手に更新しません。
3. `save_dxf_settings`: レビューした `draft_id` を保存。既存ファイルを置き換える場合は
   `.bak-<識別子>` を作成します。レビュー後にモデル・KJN・設定が変わった場合は拒否します。
4. `preview_dxf_drawings` → `save_dxf_drawings`: 保存済み設定で図面を出力。
   レビュー後に設定ファイルが変わった場合も保存を拒否します。

`settings_path` と `kjn_path` はサーバールート内の相対パスです。
`proposal_id` にも対応します。未保存GUIモデルは先に `gui_get_model` で取得します。
`preview_dxf_drawings.options.settings` に設定を直接渡して試作することもできます。
直接指定と `settings_path` の併用はできません。

## 検査

- 不明なJSON項目、バージョン違い、座標配列長・非有限値、4 MiB超過。
- 部材・断面・階・通り・注記参照の不整合、重複ID、トポロジー変更。
- INP/LST形状不一致、形状・材料・配筋定義の不足、未対応の断面図。
- GL未確定、未確認の自動推定、元モデル・LST・KJNの改訂。

エラーがあれば出力を中止します。警告はDXF出力レポートの
`drawing_issues`（コード・対象番号付き）にも記録します。

## 第②段階: 製図スタイルと図枠

`style` / `sheet` に線幅・線種・文字寸法・図枠・図番・改訂欄・回転を保存できます。
[製図スタイルと図枠の仕様](dxf-drafting.md) を参照してください。
既存の①の設定はそのまま使用でき、省略した項目は標準値になります。
