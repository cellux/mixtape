;;; mixtape.el --- Major mode for Mixtape DSL -*- lexical-binding: t; -*-

;; Author: ChatGPT
;; Version: 0.1
;; Package-Requires: ((emacs "24.4"))
;; Keywords: languages, audio
;; URL: https://github.com/cellux/mixtape

;;; Commentary:
;;
;; Simple major mode for editing Mixtape .tape files.
;; Highlighting is driven purely by regular expressions derived from
;; the language syntax described in README.md, assets/prelude.tape and tests/*.tape.
;; No built-in word lists are used.

;;; Code:

(defgroup mixtape nil
  "Major mode for the Mixtape DSL."
  :group 'languages)

(defconst mixtape--number-regexp
  ;; Numeric literals: integers, floats, optional ratio part, optional time suffix (s|b|t|p).
  "-?[0-9]+(?:\\.[0-9]+)?(?:/[0-9]+(?:\\.[0-9]+)?)?[sbtp]?"
  "Regexp matching Mixtape numeric literals, ratios, and time-suffixed numbers.")

(defconst mixtape--midi-regexp
  ;; MIDI note literals like c-4, c#4, b-9, etc.
  "[A-Ga-g](?:#|-)?-?[0-9]+"
  "Regexp matching MIDI note tokens.")

(defconst mixtape--env-regexp
  ;; Environment access shorthand: :foo, @bar, >baz
  "[:@>][A-Za-z0-9_./+-]+"
  "Regexp matching env shorthand tokens.")

(defconst mixtape--symbol-regexp
  ;; Bare symbols/words (excluding leading punctuation forms handled separately).
  "[A-Za-z~][A-Za-z0-9_./+-]*"
  "Regexp matching generic Mixtape words.")

(defconst mixtape-font-lock-keywords
  `((,mixtape--number-regexp . font-lock-constant-face)
    (,mixtape--midi-regexp . font-lock-constant-face)
    (,mixtape--env-regexp . font-lock-variable-name-face)
    ("[][{}()]" . font-lock-builtin-face)
    (";.*$" . font-lock-comment-face)
    ;; Strings are handled by syntax table, but keep a rule for robustness.
    ;; TODO: consider supporting block comments if the DSL ever adds them.
    ("\"[^\"]*\"" . font-lock-string-face)
    ;; Symbols/words fallback
    (,mixtape--symbol-regexp . font-lock-function-name-face))
  "Font-lock rules for `mixtape-mode'.")

(defvar mixtape-mode-syntax-table
  (let ((st (make-syntax-table)))
    ;; Line comments start with ;
    (modify-syntax-entry ?\; "<" st)
    (modify-syntax-entry ?\n ">" st)
    ;; Strings delimited by double quotes
    (modify-syntax-entry ?\" "\"" st)
    ;; Treat underscore and slash as word constituents for tokens
    (modify-syntax-entry ?_ "w" st)
    (modify-syntax-entry ?/ "w" st)
    (modify-syntax-entry ?+ "w" st)
    (modify-syntax-entry ?- "w" st)
    (modify-syntax-entry ?. "w" st)
    (modify-syntax-entry ?: "w" st)
    (modify-syntax-entry ?@ "w" st)
    (modify-syntax-entry ?> "w" st)
    (modify-syntax-entry ?~ "w" st)
    st)
  "Syntax table for `mixtape-mode'.")

(define-derived-mode mixtape-mode prog-mode "Mixtape"
  "Major mode for the Mixtape DSL."
  :syntax-table mixtape-mode-syntax-table
  (setq-local font-lock-defaults
              '(mixtape-font-lock-keywords
                ;; keywords-only?; case-fold?; syntax-alist?; syntax-begin?; keywords-repl?
                nil nil ((?_ . "w"))
                nil))
  ;; Comments
  (setq-local comment-start "; ")
  (setq-local comment-start-skip ";+\\s-*")
  (setq-local comment-end "")
  ;; Strings: rely on syntax table
  )

(provide 'mixtape)

;;; mixtape.el ends here
