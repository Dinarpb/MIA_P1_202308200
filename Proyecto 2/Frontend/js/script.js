let sistemaActual = [];

// --- UTILIDADES DE UI ---
function openModal(id) { document.getElementById(id).style.display = 'flex'; }
function closeModal(id) { document.getElementById(id).style.display = 'none'; }

// MANEJO DE VISTAS (Aquí cambiamos entre Discos, Físico y EXT2)
function changeView(view) {
    document.getElementById('view-disks').style.display = view === 'disks' ? 'block' : 'none';
    document.getElementById('view-explorer-fisico').style.display = view === 'explorer-fisico' ? 'block' : 'none';
    document.getElementById('view-explorer-ext2').style.display = view === 'explorer-ext2' ? 'block' : 'none';

    if (view === 'disks') refreshUI();
    if (view === 'explorer-fisico') cargarCarpetaFisica('/home/dinaarpb');
    if (view === 'explorer-ext2') cargarCarpetaExt2('/');
}

function toggleFdiskFields() {
    const op = document.getElementById('fdOp').value;
    document.getElementById('fdiskCreateFields').style.display = op === 'create' ? 'block' : 'none';
    document.getElementById('fdiskDeleteFields').style.display = op === 'delete' ? 'block' : 'none';
    document.getElementById('fdiskAddFields').style.display = op === 'add' ? 'block' : 'none';
}
function toggleOpFields() {
    const op = document.getElementById('opType').value;
    document.getElementById('opDestFields').style.display = (op === 'copy' || op === 'move') ? 'block' : 'none';
    document.getElementById('opNameFields').style.display = op === 'rename' ? 'block' : 'none';
    document.getElementById('opContFields').style.display = op === 'edit' ? 'block' : 'none';
}

// --- CONEXIÓN API COMANDOS ---
async function ejecutar(comandoStr) {
    const consola = document.getElementById('consola');
    consola.innerHTML += `\n> ${comandoStr}`;

    try {
        const response = await fetch('http://localhost:3000/ejecutar', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ comando: comandoStr })
        });
        const data = await response.json();
        consola.innerHTML += `\n${data.output || "Ejecutado correctamente."}\n`;

        // Refrescamos vistas si aplica
        if (document.getElementById('view-disks').style.display !== 'none') {
            refreshUI();
        }
        if (document.getElementById('view-explorer-ext2').style.display !== 'none') {
            cargarCarpetaExt2(document.getElementById('rutaExt2Actual').innerText);
        }
    } catch (error) {
        consola.innerHTML += `\n[ERROR] No se pudo conectar a la API.\n`;
    }
    consola.scrollTop = consola.scrollHeight;
}

// --- MANEJO DE FORMULARIOS ---
function handleForm(e, command) {
    e.preventDefault();
    let cmdStr = "";

    if (command === 'mkdisk') {
        cmdStr = `mkdisk -size=${document.getElementById('mkSize').value} -unit=${document.getElementById('mkUnit').value} -fit=${document.getElementById('mkFit').value} -path="${document.getElementById('mkPath').value}"`;
        closeModal('mkdiskModal');
    } else if (command === 'rmdisk') {
        if (confirm("¿Seguro que deseas eliminar el disco físicamente?")) {
            cmdStr = `rmdisk -path="${document.getElementById('rmPath').value}"`;
        }
        closeModal('rmdiskModal');
    } else if (command === 'fdisk') {
        const path = document.getElementById('fdPath').value;
        const name = document.getElementById('fdName').value;
        const op = document.getElementById('fdOp').value;
        if (op === 'create') cmdStr = `fdisk -size=${document.getElementById('fdSize').value} -unit=${document.getElementById('fdUnit').value} -path="${path}" -type=${document.getElementById('fdType').value} -fit=${document.getElementById('fdFit').value} -name="${name}"`;
        else if (op === 'delete') cmdStr = `fdisk -delete=${document.getElementById('fdDelete').value} -name="${name}" -path="${path}"`;
        else if (op === 'add') cmdStr = `fdisk -add=${document.getElementById('fdAdd').value} -unit=${document.getElementById('fdAddUnit').value} -name="${name}" -path="${path}"`;
        closeModal('fdiskModal');
    } else if (command === 'mkfile') {
        const path = document.getElementById('fileMkPath').value;
        const size = document.getElementById('fileMkSize').value;
        const cont = document.getElementById('fileMkCont').value;
        const r = document.getElementById('fileMkR').checked;
        cmdStr = `mkfile -path="${path}"`;
        if (size) cmdStr += ` -size=${size}`;
        if (cont) cmdStr += ` -cont="${cont}"`;
        if (r) cmdStr += ` -r`;
        closeModal('mkfileModal');
    } else if (command === 'mkdir') {
        const path = document.getElementById('dirMkPath').value;
        const p = document.getElementById('dirMkP').checked;
        cmdStr = `mkdir -path="${path}"`;
        if (p) cmdStr += ` -p`;
        closeModal('mkdirModal');
    } else if (command === 'fileops') {
        const op = document.getElementById('opType').value;
        const path = document.getElementById('opPath').value;
        if (op === 'copy') cmdStr = `copy -path="${path}" -destino="${document.getElementById('opDest').value}"`;
        else if (op === 'move') cmdStr = `move -path="${path}" -destino="${document.getElementById('opDest').value}"`;
        else if (op === 'rename') cmdStr = `rename -path="${path}" -name="${document.getElementById('opName').value}"`;
        else if (op === 'edit') cmdStr = `edit -path="${path}" -contenido="${document.getElementById('opCont').value}"`;
        else if (op === 'remove') cmdStr = `remove -path="${path}"`;
        closeModal('fileOpsModal');
    }
    if (cmdStr) ejecutar(cmdStr);
}

async function ejecutarAdmin(tipo) {
    let cmd = "";
    if (tipo === 'user') {
        const op = document.getElementById('usrOp').value;
        const u = document.getElementById('usrName').value;
        const p = document.getElementById('usrPass').value;
        const g = document.getElementById('usrGrp').value;
        if (op === 'mkusr') cmd = `mkusr -user="${u}" -pass="${p}" -grp="${g}"`;
        else if (op === 'rmusr') cmd = `rmusr -user="${u}"`;
        else if (op === 'chgrp') cmd = `chgrp -user="${u}" -grp="${g}"`;
        closeModal('adminUsersModal');
    } else {
        const op = document.getElementById('grpOp').value;
        const g = document.getElementById('grpName').value;
        if (op === 'mkgrp') cmd = `mkgrp -name="${g}"`;
        else if (op === 'rmgrp') cmd = `rmgrp -name="${g}"`;
        closeModal('adminGroupsModal');
    }
    if (cmd) {
        await ejecutar(cmd);
        cargarCarpetaExt2(document.getElementById('rutaExt2Actual').innerText);
    }
}

// --- RENDERIZADO DISCOS ---
async function refreshUI() {
    const diskGrid = document.getElementById('diskGrid');
    diskGrid.innerHTML = "<p style='padding: 20px;'>Leyendo discos en el servidor...</p>";
    document.getElementById('sectionTitle').innerHTML = `Discos Disponibles`;
    document.getElementById('btnPartition').style.display = 'none';

    try {
        // Fetch de la vista de discos
        const res = await fetch(`http://localhost:3000/sistema?t=${Date.now()}`);
        sistemaActual = await res.json();
        diskGrid.innerHTML = "";

        if (!sistemaActual || sistemaActual.length === 0) {
            diskGrid.innerHTML = "<p style='padding: 20px; color: var(--warning);'>No se encontraron discos (.dsk / .mia).</p>";
            return;
        }
        sistemaActual.forEach((disco, index) => {
            const nombreDisco = disco.path.split('/').pop();
            diskGrid.innerHTML += `
                        <div class="grid-item" onclick="selectDisk(${index}, this)">
                            <div class="icon">🖴</div>
                            <div class="item-name">${nombreDisco}</div>
                            <div class="item-sub" style="font-size: 10px;">${disco.path}</div>
                        </div>
                    `;
        });
    } catch (e) {
        diskGrid.innerHTML = "<p style='padding: 20px; color: var(--danger);'>Error al conectar con el backend.</p>";
    }
}

function selectDisk(index, element) {
    const disco = sistemaActual[index];
    document.getElementById('fdPath').value = disco.path;
    document.getElementById('btnPartition').style.display = 'block';

    const nombreDisco = disco.path.split('/').pop();
    document.getElementById('sectionTitle').innerHTML = `Particiones en ${nombreDisco} <button onclick="refreshUI()" style="margin-left: 15px; font-size: 12px; padding: 5px 10px; background: var(--border);">⬅ Volver a Discos</button>`;

    const diskGrid = document.getElementById('diskGrid');
    diskGrid.innerHTML = "";

    if (!disco.particiones || disco.particiones.length === 0) {
        diskGrid.innerHTML = "<p style='padding: 20px;'>Este disco no tiene particiones o el MBR está vacío.</p>";
        return;
    }

    disco.particiones.forEach(part => {
        const colorIcono = part.montada ? 'var(--success)' : 'var(--cyan)';
        const txtEstado = part.montada ? `<span style="color:var(--success)">● Montada (${part.id})</span>` : `<span style="color:#888">○ Desmontada</span>`;

        diskGrid.innerHTML += `
                    <div class="grid-item" onclick="openPartActions('${disco.path}', '${part.nombre}', ${part.montada}, '${part.id}')">
                        <div class="icon" style="color: ${colorIcono};">💿</div>
                        <div class="item-name">${part.nombre}</div>
                        <div class="item-sub">${txtEstado}</div>
                    </div>
                `;
    });
}

// --- ACCIONES DE PARTICIÓN ---
function openPartActions(path, name, isMounted, mountId) {
    document.getElementById('actPath').value = path;
    document.getElementById('actName').value = name;
    document.getElementById('partTitle').innerText = `Acciones: ${name}`;
    if (isMounted) {
        document.getElementById('btnActionMount').style.display = 'none';
        document.getElementById('divMountedActions').style.display = 'flex';
        document.getElementById('actId').value = mountId;
    } else {
        document.getElementById('btnActionMount').style.display = 'block';
        document.getElementById('divMountedActions').style.display = 'none';
        document.getElementById('actId').value = '';
    }
    openModal('partActionsModal');
}

function actionMount() {
    ejecutar(`mount -path="${document.getElementById('actPath').value}" -name="${document.getElementById('actName').value}"`);
    closeModal('partActionsModal');
}

function actionUnmount() {
    ejecutar(`unmount -id=${document.getElementById('actId').value}`);
    closeModal('partActionsModal');
}

function actionMkfs() {
    if (confirm("Se borrarán los datos para aplicar el formato EXT2. ¿Continuar?")) {
        ejecutar(`mkfs -type=${document.getElementById('actFsType').value} -id=${document.getElementById('actId').value}`);
        closeModal('partActionsModal');
    }
}


// ================= EXPLORADOR FÍSICO (WSL) =================
async function cargarCarpetaFisica(ruta) {
    document.getElementById('rutaFisicaActual').innerText = ruta;
    const grid = document.getElementById('gridFisico');
    grid.innerHTML = "<p style='padding: 20px;'>Leyendo WSL...</p>";

    try {
        // Llama al endpoint de Go configurado para leer directorios físicos reales
        const response = await fetch(`http://localhost:3000/explorar-fisico?ruta=${encodeURIComponent(ruta)}`);
        const data = await response.json();

        if (!response.ok) {
            grid.innerHTML = `<p style='color: var(--danger); padding: 20px;'>${data.error}</p>`;
            return;
        }

        grid.innerHTML = "";

        if (ruta !== '/' && ruta !== '/home') {
            const rutaPadre = ruta.substring(0, ruta.lastIndexOf('/')) || '/';
            grid.innerHTML += `
                        <div class="grid-item" onclick="cargarCarpetaFisica('${rutaPadre}')">
                            <div class="icon" style="color: var(--warning);">📁</div>
                            <div class="item-name">.. (Subir nivel)</div>
                        </div>`;
        }

        if (!data.archivos || data.archivos.length === 0) {
            grid.innerHTML += "<p style='padding: 20px;'>Carpeta vacía.</p>";
            return;
        }

        data.archivos.forEach(archivo => {
            const icono = archivo.tipo === '0' ? '📁' : '📄';
            let colorIcono = archivo.tipo === '0' ? 'var(--warning)' : '#ccc';

            let onclickAction = "";
            if (archivo.tipo === '0') {
                onclickAction = `onclick="cargarCarpetaFisica('${archivo.ruta}')"`;
            } else if (archivo.nombre.endsWith('.dsk') || archivo.nombre.endsWith('.mia')) {
                colorIcono = 'var(--cyan)';
                // Opcional: Al darle clic a un disco en físico, nos manda a la vista de Discos
                onclickAction = `onclick="changeView('disks');"`;
            }

            grid.innerHTML += `
                        <div class="grid-item" ${onclickAction}>
                            <div class="icon" style="color: ${colorIcono};">${icono}</div>
                            <div class="item-name">${archivo.nombre}</div>
                        </div>`;
        });

    } catch (error) {
        grid.innerHTML = "<p style='color: var(--danger); padding: 20px;'>Error de conexión.</p>";
    }
}


// ================= EXPLORADOR EXT2 (SESIÓN) =================
async function cargarCarpetaExt2(ruta) {
    document.getElementById('rutaExt2Actual').innerText = ruta;
    const grid = document.getElementById('gridExt2');
    grid.innerHTML = "<p style='padding: 20px;'>Leyendo partición...</p>";

    try {
        const response = await fetch(`http://localhost:3000/explorar?ruta=${encodeURIComponent(ruta)}`);
        const data = await response.json();

        if (!response.ok) {
            grid.innerHTML = `<p style='color: var(--danger); padding: 20px;'>${data.error}</p>`;
            return;
        }

        grid.innerHTML = "";

        if (ruta !== '/') {
            const rutaPadre = ruta.substring(0, ruta.lastIndexOf('/')) || '/';
            grid.innerHTML += `
                        <div class="grid-item" onclick="cargarCarpetaExt2('${rutaPadre}')">
                            <div class="icon" style="color: var(--warning);">📁</div>
                            <div class="item-name">.. (Regresar)</div>
                        </div>`;
        }

        if (!data.archivos || data.archivos.length === 0) {
            grid.innerHTML += "<p style='padding: 20px;'>Carpeta vacía.</p>";
            return;
        }

        data.archivos.forEach(archivo => {
            const nombreItem = archivo.nombre.trim();

            // PARCHE DE SEGURIDAD: 
            // Si el backend falla, forzamos a que sea archivo si tiene un punto en el nombre
            // (Excluyendo las carpetas especiales '.' y '..')
            const esArchivo = archivo.tipo === '1' || (nombreItem.includes('.') && nombreItem !== '.' && nombreItem !== '..');

            const icono = esArchivo ? '📄' : '📁';
            const colorIcono = esArchivo ? 'var(--accent)' : 'var(--warning)';

            let onclickAction = "";
            if (!esArchivo) {
                // Es Carpeta: Navegamos hacia adentro
                const nuevaRuta = ruta === '/' ? `/${nombreItem}` : `${ruta}/${nombreItem}`;
                onclickAction = `onclick="cargarCarpetaExt2('${nuevaRuta}')"`;
            } else {
                // Es Archivo: Hacemos CAT para leerlo
                const rutaArchivo = ruta === '/' ? `/${nombreItem}` : `${ruta}/${nombreItem}`;
                onclickAction = `onclick="verContenido('${rutaArchivo}')"`;
            }

            grid.innerHTML += `
                        <div class="grid-item" ${onclickAction}>
                            <div class="icon" style="color: ${colorIcono};">${icono}</div>
                            <div class="item-name">${nombreItem}</div>
                        </div>`;
        });

    } catch (error) {
        grid.innerHTML = "<p style='color: var(--danger); padding: 20px;'>Error de conexión.</p>";
    }
}

// Para ver el contenido con CAT dentro del EXT2
async function verContenido(rutaArchivo) {
    await ejecutar(`cat -file1="${rutaArchivo}"`);
}

// --- ARRANQUE INICIAL ---
window.onload = () => {
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('view') === 'explorer') {
        changeView('explorer-ext2');
    } else {
        changeView('disks');
    }
};

document.getElementById('loginForm').addEventListener('submit', async function (e) {
    e.preventDefault(); // Evita que la página se recargue

    const user = document.getElementById('txtUser').value;
    const pass = document.getElementById('txtPass').value;
    const id = document.getElementById('txtId').value;
    const mensajeDiv = document.getElementById('mensaje');

    mensajeDiv.style.color = "var(--text-main)";
    mensajeDiv.innerText = "Conectando...";

    // Armamos el comando de texto exacto que tu analizador espera
    const comandoLogin = `login -user=${user} -pass=${pass} -id=${id}`;

    try {
        // Enviamos el comando a tu API
        const response = await fetch('http://localhost:3000/ejecutar', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ comando: comandoLogin })
        });

        const data = await response.json();

        // Revisamos si la salida del backend contiene un error
        if (data.output && data.output.includes("[ERROR]")) {
            mensajeDiv.style.color = "var(--error)";
            mensajeDiv.innerText = data.output;
        } else {
            mensajeDiv.style.color = "var(--success)";
            mensajeDiv.innerText = "¡Login exitoso! Redirigiendo...";

            // Si el login es exitoso, redirigimos automáticamente a la pantalla del explorador
            setTimeout(() => {
                window.location.href = 'index.html?view=explorer';
            }, 1000);
        }
    } catch (error) {
        mensajeDiv.style.color = "var(--error)";
        mensajeDiv.innerText = "Error: No se pudo conectar con el servidor Go. ¿Está encendido?";
    }
});