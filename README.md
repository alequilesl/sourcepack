# sourcepack

Empaqueta todos los archivos de código fuente de una carpeta en un único archivo de texto, con un formato pensado para ser usado como contexto por una IA.

Cada archivo se incluye con su ruta relativa como encabezado, su contenido íntegro, y una línea divisoria entre archivos.

Todas las exclusiones e inclusiones se gestionan mediante ficheros estilo gitignore: `.gitignore` (opcional) y `.spignore` (siempre).

---

## Instalación

### Vía `go install` (recomendado)

Si tienes Go instalado, puedes instalar `sourcepack` directamente desde el repositorio:

    go install github.com/tu-usuario/sourcepack@latest

> **Nota**: A partir de Go 1.17, `go install` es la forma recomendada de instalar ejecutables. `go get` ya no se usa para este propósito.

El binario se instalará en `$GOPATH/bin` (por defecto `$HOME/go/bin` en Linux/macOS).

### Configurar el PATH

Para ejecutar `sourcepack` desde cualquier directorio sin especificar la ruta completa, debes añadir el directorio de binarios de Go a tu `PATH`.

#### Linux / macOS

Añade la siguiente línea a tu fichero de configuración de shell (`~/.bashrc`, `~/.zshrc` o `~/.profile`):

    export PATH="$PATH:$(go env GOPATH)/bin"

Luego recarga la configuración:

    source ~/.bashrc  # o ~/.zshrc, ~/.profile, etc.

Verifica que funciona:

    sourcepack -h

#### Windows

El instalador de Go en Windows normalmente ya añade `%USERPROFILE%\go\bin` al PATH. Si no es así:

1. Abre **Configuración** → **Sistema** → **Acerca de** → **Configuración avanzada del sistema**.
2. Pulsa **Variables de entorno**.
3. En **Variables de usuario**, selecciona `Path` y pulsa **Editar**.
4. Añade una nueva entrada: `%USERPROFILE%\go\bin` (o la ruta que devuelva `go env GOPATH` seguida de `\bin`).
5. Acepta y abre una nueva terminal.

Verifica:

    sourcepack -h

### Compilar desde el código fuente (alternativa)

    git clone https://github.com/tu-usuario/sourcepack.git
    cd sourcepack
    go build -o sourcepack main.go

Mueve el binario `sourcepack` a un directorio de tu `PATH` (ej: `/usr/local/bin` en Linux/macOS).

---

## Ayuda

    Uso: sourcepack [opciones] <path_carpeta>

    Opciones:
      -o string
            Archivo de salida (default "output.txt")
      -no-gitignore
            No leer .gitignore (el .spignore se sigue aplicando siempre)

---

## Parámetros

| Flag            | Default      | Descripción |
|-----------------|--------------|-------------|
| `<path>`        | _(requerido)_| Directorio raíz del proyecto a empaquetar. |
| `-o`            | `output.txt` | Ruta del archivo de salida. |
| `-no-gitignore` | `false`      | Si se pasa, no se lee el `.gitignore` de la raíz. El `.spignore` se sigue aplicando siempre. |

### Comportamiento por defecto

- Se lee `.spignore` de la raíz si existe (misma sintaxis que `.gitignore`). Siempre se aplica.
- Se lee `.gitignore` de la raíz si existe, salvo que se pase `-no-gitignore`.
- Los archivos binarios (con bytes nulos) se omiten automáticamente.
- Los archivos se emiten en orden alfabético por ruta relativa.
- Las rutas se normalizan con separador `/`.
- **No hay exclusiones hardcodeadas**: todo lo que se excluye proviene de `.spignore` o `.gitignore`. Si necesitas ignorar `node_modules`, `.git`, `build`, etc., añádelos a tu `.spignore`.

---

## Ficheros de exclusión

Las exclusiones se definen con la **misma sintaxis que `.gitignore`**:

- Comentarios con `#`.
- Líneas vacías ignoradas.
- Negaciones con `!` para volver a incluir algo previamente excluido.
- Patrones solo para directorios si terminan en `/`.
- Patrones anclados a la raíz si empiezan por `/`.
- Comodines `*`, `?`, `[...]` y `**`.

### `.spignore`

Fichero opcional en la raíz del proyecto. Se aplica **siempre** (no se puede desactivar). Útil para excluir ficheros que, aunque estén versionados, no quieres incluir en el contexto de la IA, o para filtrar por extensión.

> ⚠️ No hay carpetas ignoradas por defecto. Si no quieres que se incluyan `.git/`, `node_modules/`, `build/`, etc., debes excluirlos aquí o dejar que lo haga el `.gitignore` del proyecto (que suele incluirlos).

Ejemplo:

    # Tests privados que no quiero compartir
    src/test/private

    # Documentación generada
    docs/api/swagger.json

    # Logs
    *.log

    # Excepción: sí quiero este log concreto
    !important.log

    # Filtrar por extensión: excluir todo lo que no sea código
    *.png
    *.jpg
    *.zip
    *.pdf
    *.lock

### `.gitignore`

Se lee automáticamente de la raíz si existe, para respetar las mismas exclusiones que git. Se puede desactivar con `-no-gitignore`.

### Precedencia

Ambos ficheros se combinan. Si una ruta coincide con un patrón de cualquiera de los dos, se excluye (salvo que una negación `!` posterior lo revierta). El orden de aplicación es: primero `.spignore`, luego `.gitignore`.

---

## Ejemplos

    # Uso básico (aplica .spignore y .gitignore si existen)
    sourcepack ./mi_proyecto

    # Archivo de salida personalizado
    sourcepack -o contexto.txt ./mi_proyecto

    # Ignorar el .gitignore (solo .spignore)
    sourcepack -no-gitignore -o contexto.txt ./mi_proyecto

    # Combinado
    sourcepack -o contexto.txt -no-gitignore ./monorepo

---

## Formato de salida

    path/relativo/del_archivo
    contenido íntegro del archivo

    --------------------------------------------------------------------------------
    path/relativo/del_siguiente_archivo
    contenido íntegro del siguiente archivo