# Magento Static Content Deployer (Go)

Experimental!! A high-performance static content deployment tool written in Go that significantly accelerates Magento 2 static asset deployment by leveraging true parallelization and efficient file I/O.

## Performance

On this project, deployment improved from **~115 seconds** (Magento native) to **~0.3-0.5 seconds** for the frontend theme deployment:

- **Vendor/Hyva/frontend**: 11,126 files deployed in 0.3 seconds
- **Throughput**: ~40,000 files/second
- **Speedup**: **230-380x faster** than PHP implementation

## Why It's Faster

1. **Native Parallelization**: Go's goroutines handle true concurrent I/O across multiple CPU cores
2. **Low Overhead**: No full Magento bootstrap, no dependency injection container, no database access
3. **Efficient I/O**: Optimized file copying with buffered I/O and minimal memory allocation
4. **Minimal Compilation**: Only compiles email CSS (using PHP's wikimedia/less.php); main theme CSS handled by npm build

## Installation

### Build from Source

```bash
cd tools/magento2-static-deploy
go build -o magento2-static-deploy main.go watcher.go less.go less_preprocessor.go
```

### Requirements

- Go 1.21 or later
- PHP available in PATH (uses Magento's `wikimedia/less.php` for email CSS compilation)

## Usage

### Basic Usage

Deploy Vendor/Hyva theme to frontend area:

```bash
./tools/magento2-static-deploy/magento2-static-deploy \
  -root . \
  -locales nl_NL \
  -themes Vendor/Hyva \
  -areas frontend
```

### With Verbose Output

```bash
./tools/magento2-static-deploy/magento2-static-deploy \
  -root . \
  -locales nl_NL,en_US \
  -themes Vendor/Hyva \
  -areas frontend \
  -jobs 8 \
  -v
```

### Note on Admin Themes

By default, only `frontend` area is deployed. This is because:
- Admin themes (Magento/backend, MageOS/m137-admin-theme) are part of Magento core
- They don't typically need custom deployment unless you have custom admin theme
- If deployment encounters a missing theme, it gracefully skips it

To deploy admin themes if they exist:
```bash
./magento2-static-deploy -areas frontend,adminhtml -v
```

### All Options

```
  -root string
        Path to Magento root directory (default ".")

  -locales string
        Comma-separated locales (default "nl_NL")
        Example: nl_NL,en_US,de_DE

  -themes string
        Comma-separated themes (default "Vendor/Hyva")
        Example: Vendor/Hyva,Magento/blank,Hyva/reset

  -areas string
        Comma-separated areas (default "frontend,adminhtml")
        Options: frontend, adminhtml

  -jobs int
        Number of parallel jobs (default 0 = auto-detect CPU count)
        Use -jobs 1 for sequential processing

  -strategy string
        Deployment strategy (default "quick")
        Note: Currently only copies files; strategy is informational

  -force
        Force deployment even if files exist (always copies)

  -content-version string
        Static content version (default: auto-generate timestamp)
        Use this to reuse the same version across multiple deployments

  -v    Verbose output showing per-deployment progress
```

## Examples

### Deploy Single Locale/Theme

```bash
./magento2-static-deploy -root /var/www/magento -locales nl_NL -themes Vendor/Hyva -areas frontend
```

### Deploy Multiple Locales and Themes

```bash
./magento2-static-deploy \
  -locales nl_NL,en_US,de_DE \
  -themes Vendor/Hyva,Magento/blank \
  -areas frontend
```

### Sequential Processing (1 Job)

```bash
./magento2-static-deploy -jobs 1 -v
```

### Full Admin + Frontend Deployment

```bash
./magento2-static-deploy \
  -locales nl_NL \
  -themes Vendor/Hyva \
  -areas frontend,adminhtml
```

### Reuse Content Version (for Split Deployments)

When splitting deployments across multiple runs (e.g., deploying different locales or themes in parallel), you can reuse the same content version:

```bash
# First deployment
./magento2-static-deploy \
  -locales nl_NL \
  -themes Vendor/Hyva \
  -content-version 1234567890

# Second deployment with the same version
./magento2-static-deploy \
  -locales en_US,de_DE \
  -themes Vendor/Hyva \
  -content-version 1234567890
```

This is useful for deployment tools like [Deployer](https://github.com/deployphp/deployer) or Hypernode Deploy that optimize deployments by splitting locale-theme combinations across multiple processes.

## What It Does

1. Creates combinations of (locale, theme, area)
2. For each combination:
   - Verifies source theme directory exists
   - Creates destination directory in `pub/static`
   - Recursively copies all files from source to destination
   - Counts files deployed

3. Processes jobs in parallel using goroutines
4. Reports results with timing and throughput metrics

## What It Doesn't Do (Yet)

This version performs file copying plus email CSS compilation. The following are handled separately:

- **Full LESS/SCSS Compilation**: Done by Hyva theme's npm build process (email CSS is compiled by this tool using PHP)
- **JavaScript Minification**: Done by npm/webpack
- **CSS Minification**: Done by build tools (email CSS is minified by wikimedia/less.php)
- **Symlink Fallback**: Not implemented
- **Admin Theme Deployment**: Skipped if theme doesn't exist (Magento core themes don't need custom deployment)
- **Vendor Theme Path Resolution**: Gracefully skips themes not found in app/design or vendor paths

## Comparison with Magento CLI

### Magento `setup:static-content:deploy`
- Requires full Magento bootstrap
- Single-threaded or limited parallelization
- PHP overhead for each file
- ~115 seconds for 28,500 files

### Go Tool
- Direct file operations
- True goroutine-based parallelization
- Minimal overhead per file
- ~0.3-0.5 seconds for 11,126 files

## Typical Workflow

1. **Development**: Use Hyva theme's npm build and cache-clean watch
   ```bash
   npm --prefix app/design/frontend/Vendor/Hyva/web/tailwind run dev
   ```

2. **Deployment Prep**: Run this tool to stage static files
   ```bash
   ./magento2-static-deploy -v
   ```

3. **Cache Clear** (if needed):
   ```bash
   bin/magento cache:clean
   ```

## Limitations

### Hyva Themes Only

**This tool is designed specifically for Hyva-based themes** and will not produce identical output to Magento's native `setup:static-content:deploy` for Luma/Blank themes.

#### What's Missing for Luma Support

Magento's native static deploy performs several compilation and generation steps that this tool does not:

| Feature | Magento Native | This Tool |
|---------|---------------|-----------|
| File copying | ✅ | ✅ |
| Email CSS compilation | ✅ | ✅ (via wikimedia/less.php) |
| LESS → CSS compilation (full) | ✅ | ❌ |
| RequireJS config merging | ✅ | ❌ |
| JS translation generation | ✅ | ❌ |
| JavaScript bundling | ✅ | ❌ |
| SRI hash generation | ✅ | ❌ |

#### Why Not Implement Full Luma Support?

Implementing full Luma compatibility would require:

1. **Full LESS Compilation** - Compiling all theme LESS files (not just email CSS) with proper source file resolution
2. **RequireJS Config Merging** - Parsing and merging JavaScript config objects from all modules
3. **JS Translation Generation** - Reading Magento's PHP translation dictionaries and converting to JSON
4. **JavaScript Bundling** - Implementing Magento's complex bundling logic

While we now use PHP's `wikimedia/less.php` for email CSS compilation (matching Magento's behavior), implementing full Luma LESS compilation would require recreating Magento's complex source file resolution and preprocessing logic.

**For Hyva themes, none of this is needed** because:
- Hyva uses Tailwind CSS (pre-built), not LESS
- Hyva doesn't use RequireJS
- Hyva handles translations differently
- JavaScript is bundled via npm/webpack during theme build

#### Recommendation

- **Hyva themes**: Use this tool for 70-90x faster deployments
- **Luma/Blank themes**: Continue using `bin/magento setup:static-content:deploy`

### Current Capabilities

- ✅ Fast parallel file copying
- ✅ Multi-locale/theme/area support
- ✅ Theme module overrides (`app/design/{area}/{vendor}/{theme}/{Module}/web/`)
- ✅ Vendor module web assets
- ✅ Library files (`lib/web/`)
- ✅ Content version management
- ✅ Verbose progress reporting
- ✅ Email CSS compilation (email.css, email-inline.css, email-fonts.css)

### Not Implemented

- ❌ Full LESS/SCSS compilation (use npm for Hyva)
- ❌ RequireJS config merging
- ❌ JavaScript bundling
- ❌ JS translation generation
- ❌ Symlink fallback strategy
- ❌ Incremental deployment detection

### Email CSS Differences

The email CSS output is nearly identical to Magento's native output since we use the same PHP LESS compiler (`wikimedia/less.php`). Minor differences may occur:

1. **Font family**: The Go binary correctly resolves theme variable inheritance (e.g., `'Open Sans'` from Blank theme), while Magento's preprocessing may produce different results depending on the theme hierarchy.

2. **URL placeholders**: Both use the correct `{{base_url_path}}` format for email-fonts.css imports.

These differences are functionally equivalent and should not affect email rendering.

## Development

### Code Structure

- `main.go`: CLI interface, orchestration logic
- `watcher.go`: File change detection (for future watch mode)
- `less.go`: LESS to CSS compilation using PHP's wikimedia/less.php (same as Magento)
- `less_preprocessor.go`: Magento-style LESS preprocessing (@magento_import, source staging)

### Building

```bash
go build -o magento2-static-deploy main.go watcher.go less.go less_preprocessor.go
```

### Performance Profiling

```bash
time ./magento2-static-deploy -v
```

## Integration with Existing Workflow

Since this tool only copies files, it integrates well with existing Magento setups:

1. Hyva theme builds are done via npm
2. Static files are copied to pub/static by this tool
3. Cache can be cleared separately as needed

## CI/CD Integration

### GitLab CI/CD

If you're using a GitLab CI/CD pipeline with a `setup-static-content-deploy` job, you can override it to use this Go binary for significantly faster frontend deployments:

```yaml
# Override setup-static-content-deploy to use Go binary for frontend (10x faster)
setup-static-content-deploy:
  stage: build
  script:
    - source ~/.nvm/nvm.sh
    - nvm use ${NODE_VERSION:---lts} || nvm install ${NODE_VERSION:---lts}
    # Build Hyva theme assets
    - if [ ! -z $THEME_PATH ]; then npm --prefix $THEME_PATH ci --no-audit; fi
    - if [ ! -z $THEME_PATH ]; then NODE_ENV=production npm --prefix $THEME_PATH run build-prod; fi
    # Download and use Go binary for frontend static content (much faster)
    - echo "Downloading magento2-static-deploy binary..."
    - curl -sL -o /tmp/magento2-static-deploy https://github.com/elgentos/magento2-static-deploy/releases/latest/download/magento2-static-deploy-linux-amd64
    - chmod +x /tmp/magento2-static-deploy
    # Deploy frontend static content using Go binary (uses PHP for email CSS compilation)
    - echo "Deploying frontend static content using Go binary..."
    - /tmp/magento2-static-deploy -root . -themes ${THEMES} -locales $(echo ${STATIC_LOCALES} | tr ' ' ',') -areas frontend -v
    # Deploy admin static content using Magento CLI (Go binary is Hyva-focused)
    - php bin/magento setup:static-content:deploy -f -j ${JOB_CONCURRENCY:-$(nproc)} --area adminhtml ${STATIC_ADMIN_LOCALES:-"nl_NL en_US"}
```

This approach uses the Go binary for frontend (Hyva) themes while falling back to Magento's native CLI for adminhtml, which may require RequireJS config merging and other Luma-specific processing.

### Deployer (Pull Approach)

If you use [Deployer](https://deployer.org/) with a pull-based deployment strategy, you can override the `magento:deploy:assets` task:

```php
task('magento:deploy:assets', function () {
    // Deploy adminhtml using Magento CLI
    invoke('magento:deploy:assets:adminhtml');

    // Download the Go binary
    within("{{release_or_current_path}}", function () {
        run('curl -sL -o /tmp/magento2-static-deploy https://github.com/elgentos/magento2-static-deploy/releases/latest/download/magento2-static-deploy-linux-amd64');
        run('chmod +x /tmp/magento2-static-deploy');
    });

    // Deploy frontend themes using Go binary
    $themes = get('magento_themes');
    foreach ($themes as $theme => $locales) {
        within("{{release_or_current_path}}", function () use ($theme, $locales) {
            run('echo "Deploying static content for theme ' . $theme . ' and locales: ' . $locales . '"');
            run('/tmp/magento2-static-deploy -root . -themes ' . $theme . ' -locales ' . join(',', explode(' ', $locales)) . ' -areas frontend -v');
        });
    }
});
```

Make sure your `magento_themes` configuration is set up in your `deploy.php`:

```php
set('magento_themes', [
    'Vendor/Hyva' => 'nl_NL en_US',
]);
```
