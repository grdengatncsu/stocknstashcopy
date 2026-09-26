# Prototype 1 Evaluation Set and Image-Source Plan

## Purpose

This document defines the small supported grocery set and image-evaluation plan for Prototype 1.

The goal is to evaluate a limited, representative set of groceries under the actual Stock 'n Stash camera geometry. It is not a plan to support an unrestricted grocery catalog or to collect a production-scale training dataset.

The current pretrained baseline is YOLOv8n. It proves the three-camera recognition interface works, but it does not provide reliable exact grocery-SKU recognition by itself. Prototype 1 therefore separates:

- **general-category recognition** for visually recognizable produce or generic object classes, and
- **exact-SKU recognition** for selected packaged products where exact identity is required.

If the system cannot meet the required recognition level with sufficient confidence, the result must be returned as `unknown` for user review rather than being forced into the nearest supported label.

## Initial Prototype 1 Grocery Set

The initial set contains 12 products. The exact packaged SKU and UPC must be taken from the physical unit used for evaluation so package-size variants are not accidentally mixed.

| ID | Product | Package type | Required recognition level | Product-database lookup | Visual recognition required | UPC / barcode |
| --- | --- | --- | --- | --- | --- | --- |
| `apple` | Apple | loose produce | General category | No | Yes | N/A |
| `banana` | Banana | loose produce | General category | No | Yes | N/A |
| `orange` | Orange | loose produce | General category | No | Yes | N/A |
| `coca-cola-can` | Coca-Cola can, exact size used by team | can | Exact SKU | Yes | Yes | Record from physical unit |
| `campbells-chicken-noodle` | Campbell's Chicken Noodle Soup, exact size used by team | can | Exact SKU | Yes | Yes | Record from physical unit |
| `heinz-ketchup` | Heinz Tomato Ketchup, exact size used by team | bottle | Exact SKU | Yes | Yes | Record from physical unit |
| `jif-creamy` | Jif Creamy Peanut Butter, exact size used by team | jar | Exact SKU | Yes | Yes | Record from physical unit |
| `cheerios` | Cheerios, exact box size used by team | box | Exact SKU | Yes | Yes | Record from physical unit |
| `barilla-spaghetti` | Barilla spaghetti, exact box size used by team | box | Exact SKU | Yes | Yes | Record from physical unit |
| `lays-classic` | Lay's Classic Potato Chips, exact bag size used by team | bag | Exact SKU | Yes | Yes | Record from physical unit |
| `chobani-yogurt` | Chobani Greek Yogurt, exact flavor/size used by team | cup | Exact SKU | Yes | Yes | Record from physical unit |
| `tropicana-orange-juice` | Tropicana Orange Juice, exact size used by team | bottle/carton | Exact SKU | Yes | Yes | Record from physical unit |

### Why this set

The set intentionally includes:

- loose produce with no barcode,
- rigid boxes,
- flexible bags,
- bottles and jars,
- metal cans,
- a small cup/container,
- objects with glossy surfaces that can produce glare,
- products with visually similar package shapes but different identities.

This keeps the evaluation small enough for a senior-design prototype while exercising the package types and recognition conditions that are expected during the final demonstration.

## Product Metadata and Database Lookup

For packaged products:

1. Record the barcode/UPC directly from the exact physical item used in testing.
2. Record the brand, product name, flavor/variant, package size, and UPC in the evaluation metadata.
3. Check whether that exact UPC exists in Open Food Facts.
4. If Open Food Facts contains the product, record the product entry as the preferred external metadata/image source.
5. If the exact UPC is missing, keep the physical UPC in local metadata and mark the database lookup as unavailable rather than substituting a different size or variant.

Open Food Facts is the preferred product-database source for Prototype 1 because it can provide barcode-linked product metadata and product images. Manufacturer product pages may be used as a secondary visual reference when needed, but they do not replace the physical-product ground truth.

Loose produce does not require database lookup for Prototype 1.

## Pretrained-Model Approach

The existing baseline uses YOLOv8n with a confidence threshold of `0.50`.

For issue #5 evaluation planning:

- YOLOv8n remains the initial pipeline baseline.
- Its generic labels are treated as evidence that the detection pipeline works, not as proof of exact-SKU recognition.
- Produce categories such as apple, banana, and orange can be evaluated directly when the model has an appropriate class.
- Exact packaged SKUs must not be accepted from a generic class label alone.
- A packaged product that does not meet the required exact-SKU recognition path is returned as `unknown`.
- Fine-tuning or a separate product-classification stage is considered only after the baseline evaluation shows that it is necessary.

The deployed Hailo model may differ from YOLOv8n, but it must preserve the same supported-product taxonomy and `unknown` behavior.

## Image Sources

### Public / database images

Public images are development references, not the final holdout set.

Preferred sources:

- Open Food Facts product images for exact packaged products whose UPC exists in the database.
- Public benchmark or open-dataset images for generic produce categories when licensing and class definitions are appropriate.
- Manufacturer product pages only as secondary references when a database image is unavailable.

For every externally sourced image collection, record:

- source name,
- source URL or product identifier,
- product ID from this document,
- date accessed,
- whether the image is used for development, training, or reference only,
- any applicable license or usage restriction.

Do not commit large image collections to Git. Dataset files remain outside the repository under the ignored `datasets/` path.

## Actual Three-Camera Evaluation Capture

Final evaluation images must be captured using the real mounted Stock 'n Stash cameras.

Logical camera names must match the recognition baseline:

- `overhead`
- `rear_left`
- `front_right`

Each capture event receives one `scene_id`. The three images from the same physical scene share that `scene_id` so they can later be associated and evaluated as one synchronized observation.

### Minimum capture coverage

For each supported product, capture scenes that include:

- normal front-facing presentation,
- approximately 90-degree rotation,
- approximately 180-degree rotation or rear-facing package view when meaningful,
- off-axis / diagonal orientation,
- center of platform,
- near at least one platform edge,
- partial occlusion by another grocery item,
- glare or specular reflection when applicable,
- at least one multi-item scene.

Across the full set, the evaluation must include:

- bags,
- boxes,
- bottles/jars,
- cans,
- loose produce,
- at least one repeated-category or repeated-product scene,
- scenes containing two or more groceries at once.

The intent is to test realistic variation, not to generate many nearly identical frames.

## Image Naming and Folder Structure

Large datasets are not committed to Git.

Recommended local structure:

```text
datasets/
└── prototype1/
    ├── metadata/
    │   ├── products.csv
    │   └── scenes.csv
    ├── public/
    │   └── <product_id>/
    ├── captures/
    │   ├── development/
    │   └── holdout/
    └── notes/
```

Recommended captured-image filename:

```text
<scene_id>__<product-or-scene-label>__<camera>__<condition>.jpg
```

Example:

```text
S0042__cheerios__rear_left__rot90-glare.jpg
```

Required camera token values are exactly:

- `overhead`
- `rear_left`
- `front_right`

For multi-item scenes, use a scene label such as `multi-apple-coke-cheerios` rather than trying to encode every ground-truth field in the filename.

Ground truth belongs in `scenes.csv`, not only in filenames.

## Evaluation and Holdout Rules

### Development images

Development images may be used to:

- verify image loading,
- debug camera capture,
- tune confidence thresholds,
- inspect class behavior,
- decide whether fine-tuning is necessary.

Public/database images belong to the development side unless a separate licensed benchmark protocol is explicitly adopted.

### Holdout images

The final holdout set must:

- come from the actual mounted three-camera system,
- be captured after the main development/tuning process,
- contain scenes not previously used to tune thresholds or model behavior,
- remain untouched until final evaluation,
- keep all three views of a scene in the same split,
- avoid near-duplicate burst frames across development and holdout.

Do not randomly split frames from one capture burst across development and holdout. Split by capture session or scene to prevent leakage.

### Ground truth

For each holdout scene, record:

- products physically present,
- quantity of each product,
- expected recognition level for each product,
- required exact SKU where applicable,
- camera views in which each item is visibly observable,
- any deliberate glare, occlusion, or orientation condition.

A camera is not counted as wrong for failing to identify information that is genuinely not visible from that view. Final end-to-end evaluation should consider the fused three-camera result.

## Fine-Tuning Rules, Only If Needed

Fine-tuning is not required merely to complete this planning issue.

If baseline testing shows that fine-tuning is necessary:

1. Freeze the final holdout set before training.
2. Never train on holdout images.
3. Split by scene/capture session, not by individual near-duplicate frame.
4. Keep all three views from one scene in the same split.
5. Prefer images captured from the real prototype geometry over scraped images.
6. Record class counts and source provenance.
7. Do not silently relabel unsupported products into a supported class.
8. Re-run the untouched holdout after training and report results separately from training/validation metrics.

## Unsupported and Uncertain Products

A detection returns `unknown` when any of the following is true:

- the product is outside the Prototype 1 supported set,
- confidence is below the selected threshold,
- the model only provides a generic class when the product requires exact-SKU recognition,
- observations across cameras conflict and cannot be resolved confidently,
- the item is too occluded or visually ambiguous to meet the required recognition level.

The application can then ask the user to confirm or correct the item.

## Image-Collection Responsibilities

| Responsibility | Owner |
| --- | --- |
| Supported-product list, recognition-level rules, evaluation split, and final holdout control | Jaelynn |
| Public/database image-source collection and product-label review | Gavin |
| Three-camera capture procedure and camera-side validation | Justin and Luke |
| Fixture consistency, placement/orientation test setup, and physical scene repeatability | Jackson |
| Acquire/organize packaged products and record exact package size + UPC from each physical unit | Maxime |
| Final review of labels, metadata, and issue-completion checklist | Jaelynn + team |

## Remaining Actions Before Issue #5 Can Close

- [ ] Team confirms the proposed 12-product set.
- [ ] Exact package size/flavor is fixed for each packaged product.
- [ ] UPC/barcode is copied from each exact physical packaged product.
- [ ] Open Food Facts lookup status is recorded for each packaged product.
- [ ] Public/database source references are added to the metadata.
- [ ] Team confirms image-collection ownership.
- [ ] The plan is updated after the first mounted-camera capture if the physical setup requires a naming or folder adjustment.

Once those items are filled in, this document satisfies the planning scope of issue #5.
