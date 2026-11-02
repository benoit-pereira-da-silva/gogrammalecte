# Embedded lexical source data

The four frequency files are the original unversioned Grammalecte corpus files, recovered from Fedora’s archived Grammalecte 2.1.2 source package. They are stored as deterministic gzip streams (empty filename, zero timestamp); decompression restores every original byte. They are not replacements from newer datasets.

## Retrieval and verification

Retrieved on 2026-11-02 from:

[Fedora source archive](https://src.fedoraproject.org/repo/pkgs/rpms/grammalecte/grammalecte-2.1.2.tar.xz/sha512/13adf63f191c80cd8d3c735fdde16c8e2e1c909f306108b9ce933b5fb6f335f65748e296e3d7798c876cc59eec8fe4db4de972825ebeab3aef6d6e6866cd7aac/grammalecte-2.1.2.tar.xz).

The archive contains 86,500,708 bytes. Its SHA-512, verified against Fedora’s `sources` manifest on branch `f39` at commit `eba34d001348039a5bffa4ba7203380f5c1689ba`, is:

```text
13adf63f191c80cd8d3c735fdde16c8e2e1c909f306108b9ce933b5fb6f335f65748e296e3d7798c876cc59eec8fe4db4de972825ebeab3aef6d6e6866cd7aac
```

The [Fedora packaging repository](https://src.fedoraproject.org/rpms/grammalecte/tree/eba34d001348039a5bffa4ba7203380f5c1689ba) contains `grammalecte-generate-tarball.sh`. That script clones `http://code.grammalecte.net:8080` with `fossil clone -u`, checks out the version tag, then explicitly runs `fossil uv export` for all four files into `gc_lang/fr/dictionnaire/lexique/corpus_data/`. This establishes that the archived files originated in upstream’s unversioned corpus storage.

The archived 176-byte `_README.txt` is byte-for-byte identical to the README in the pinned reference checkout (`08511c222029b3be20bf547563315d10fb48d44b`), with SHA-256 `e7595256c8461fc0a285090bdd23b8d04d4a0dd52c1c8defd374a077439f965f`. This corroborates provenance. It does **not** establish that the data still match the latest unversioned files: upstream’s Fossil HTTP endpoint returned HTTP 503 during retrieval, and its HTTPS endpoint had a certificate hostname mismatch. Current unversioned content equality remains unverified.

The README labels Wikipedia `2026-10-20`, Wikisource `2016-10-20`, and Google Ngram 1 `2012`. These labels are reproduced as source evidence, without asserting the accuracy of the Wikipedia date.

## Original files

| Filename | Bytes | Rows | Sum of counts | SHA-256 |
|---|---:|---:|---:|---|
| `stats_frwiki.txt` | 139,335,991 | 9,749,456 | 1,642,516,410 | `324f61d55e95f883ee5e398cb08c65eb2c9cfe853ed608a6914505aeff17c03a` |
| `stats_frwikisource.txt` | 43,784,984 | 3,634,413 | 685,439,799 | `24473ae8d39fb7c5193d755be2972db222b4b530b864a7e730e3d6acf62cb500` |
| `stats_google_ngram_1.txt` | 53,200,333 | 4,014,202 | 168,120,132,966 | `5749eea15feeca050e315f5309b6ab947fa4cd58569ceec9ce99b77fad81071f` |
| `stats_litterature.txt` | 10,058,791 | 825,149 | 157,575,024 | `b89597d6ce17902b07fabb5ed57f8b33f20ca24855b6b383fee5a91dbf7b5e04` |

The gzip payloads total 94,905,369 bytes. `../testdata/corpora.json` records compressed hashes and sizes as well. Native Go tests decompress and validate every file’s byte count, SHA-256, line count, two-field structure, and total occurrence count.

## Dictionary and affixes

`FRANCAIS.dic.gz` and `FRANCAIS_7.aff.gz` contain the original files from `gc_lang/fr/dictionnaire/orthographe/` at reference revision `08511c222029b3be20bf547563315d10fb48d44b`. They are unmodified source resources used by the native lexical generator. Their attribution remains with Olivier R. and upstream contributors under the Mozilla Public License 2.0, as in the source distribution.

The corpus files contain frequency data without a separate license notice in their archived directory. This record describes their upstream packaging and preserves them as distributed; it does not assign a new license to the datasets.

`validation-tags.json` preserves the allowed tag names from the pinned `tags.py`; its source hash is recorded in `../testdata/source.json`. Thesaurus source data are reused from the French package rather than embedded twice.
