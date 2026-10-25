; demo.scm — qwen-lisp の機能デモ
; 実行: go run . examples/demo.scm

;; 再帰関数: 階乗
(define (factorial n)
  (if (= n 0)
      1
      (* n (factorial (- n 1)))))
(display (factorial 10))
(newline)

;; 再帰関数: フィボナッチ
(define (fib n)
  (if (< n 2)
      n
      (+ (fib (- n 1)) (fib (- n 2)))))
(display (fib 20))
(newline)

;; クロージャ: 状態を持つカウンター
(define (make-counter start step)
  (let ((n start))
    (lambda ()
      (let ((v n))
        (set! n (+ n step))
        v))))
(define c1 (make-counter 0 5))
(define c2 (make-counter 100 1))
(display (list (c1) (c1) (c1) (c2) (c2)))
(newline)

;; quote とリスト操作
(define xs (list 1 2 3 4 5))
(display (quote (hello world)))
(newline)
(display (append xs '(6 7)))
(newline)
(display (cons 0 xs))
(newline)

;; 高階関数 + 再帰: map
(define (map f lst)
  (if (null? lst)
      '()
      (cons (f (car lst)) (map f (cdr lst)))))
(display (map (lambda (x) (* x x)) xs))
(newline)

;; cond と可変長引数
(define (classify n)
  (cond ((< n 0) 'negative)
        ((= n 0) 'zero)
        (else 'positive)))
(display (map classify (list -2 0 7)))
(newline)

(define (average . nums)
  (/ (apply-sum nums) (length nums)))
(define (apply-sum nums)
  (if (null? nums)
      0
      (+ (car nums) (apply-sum (cdr nums)))))
(display (average 2 4 6 8))
(newline)
