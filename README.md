# qwen-lisp

Go で実装した、Scheme 風の小さな Lisp インタプリタです。
標準ライブラリのみを使用し、外部の Lisp 処理系には依存しません。

## 使い方

### REPL を起動する

```console
$ go run .
qwen-lisp — a small Scheme-style Lisp interpreter. Press Ctrl-D to exit.
lisp> (+ 1 2)
3
lisp> (define (fact n)
  ...   (if (= n 0)
  ...       1
  ...       (* n (fact (- n 1)))))
#<procedure fact>
lisp> (fact 10)
3628800
```

開き括弧が続く間は `...` プロンプトで継続入力されます。終了は Ctrl-D です。

### スクリプトを実行する

```console
$ go run . examples/demo.scm
3628800
6765
(0 5 10 100 101)
...
```

ファイル実行時は各形式の値は表示されません。出力には `display` / `newline` を使ってください。

### 式を直接評価する

```console
$ go run . -e '(+ 1 (* 2 3))'
7
```

## テスト

```console
$ go test ./...
```

## 言語仕様

### 型

| 型 | 例 |
| --- | --- |
| 整数 (64bit) | `42` `-7` |
| 真偽値 | `#t` `#f` (`true` / `false` も可) |
| シンボル | `foo` `+` `list?` |
| リスト | `()` `(1 2 3)` `(1 . 2)` |
| 手続き | `#<procedure f>` |

小数はサポートしません（`3.14` はエラーになります）。真偽判定は Scheme 流で、偽は `#f` だけで、`()` や `0` は真です。

### 特殊形式

| 形式 | 意味 |
| --- | --- |
| `(quote x)` / `'x` | 評価せずにそのまま返す |
| `(define name value)` / `(define (name params...) body...)` | 定義 |
| `(set! name value)` | 既存の変数へ代入 |
| `(if test then [else])` | 条件分岐 |
| `(lambda (params...) body...)` | 無名手続き（クロージャ） |
| `(let ((v1 e1) ...) body...)` | ローカル束縛（値は外側の環境で評価） |
| `(begin e...)` | 順番に評価して最後の値を返す |
| `(cond (t1 b1...) ... (else b...))` | 多段条件 |
| `(and e...)` / `(or e...)` | 短絡評価 |

### 主な組み込み関数

- 演算: `+` `-` `*` `/`（整数除算、2 個以上の引数、0 で割るとエラー）
- 比較: `=` `<` `>` `<=` `>=`（2 個以上なら連鎖比較、例 `(< 1 2 3)`）
- 論理: `not`
- リスト: `cons` `car` `cdr` `list` `null?` `pair?` `list?` `length` `append`
- 述語: `eq?` `equal?` `symbol?` `integer?` `procedure?`
- 出力: `display` `newline`

### プログラム例

```scheme
;; クロージャ
(define (make-counter start step)
  (let ((n start))
    (lambda () (set! n (+ n step)) n)))
(define c (make-counter 0 5))
(c) ; => 5
(c) ; => 10

;; 再帰
(define (fib n)
  (if (< n 2) n (+ (fib (- n 1)) (fib (- n 2)))))
(fib 15) ; => 610

;; 可変長引数
(define (count . args) (length args))
(count 1 2 3) ; => 3
```

## プロジェクト構成

```
main.go                 REPL とコマンドラインエントリポイント
internal/lisp/
  object.go             Lisp の値の表現 (整数・シンボル・対など)
  lexer.go              字句解析
  parser.go             S 式の構文解析
  env.go                環境 (lexical scope)
  special.go            特殊形式 (define, lambda, let ...)
  builtins.go           組み込み関数
  eval.go               木を辿る評価器
  *_test.go             単体テストと examples/demo.scm の統合テスト
examples/demo.scm       機能デモスクリプト
```

## エラー処理

- 字句・構文エラーは行番号付きで報告（例: `line 3: unclosed '('`）
- 未定義シンボル、引数の個数違い、型の不一致、0 除算、整数オーバーフローは実行時エラー
- 誤った入力や無限再帰でも REPL はクラッシュせず、エラーを表示して継続します
  （評価の深さは 10,000 で上限）

## 制約

- 整数のみ（小数・文字列リテラルなし。`display` でシンボルや数値を表示可能）
- 末尾再帰最適化なし（深い再帰は上限に達するとエラー）
- 特殊形式の名前はユーザーコードで上書き不可
