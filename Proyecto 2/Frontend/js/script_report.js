let reporteSeleccionado = 'disk';
let particiones = [];

// Cargar particiones disponibles
async function cargarParticiones() {
    const select = document.getElementById('particionSelect');
    try {
        console.log('Iniciando carga de particiones...');

        const response = await fetch('http://localhost:3000/sistema');
        console.log('Response status:', response.status);

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const discos = await response.json();
        console.log('Discos recibidos:', discos);

        select.innerHTML = '';
        particiones = [];

        if (!Array.isArray(discos)) {
            console.warn('Respuesta no es un array:', typeof discos);
            select.innerHTML = '<option value="">Error: respuesta inválida</option>';
            return;
        }

        let partidasEncontradas = 0;

        for (const disco of discos) {
            console.log('Procesando disco:', disco);

            if (!disco.particiones) {
                console.log('Disco sin campo particiones');
                continue;
            }

            if (!Array.isArray(disco.particiones)) {
                console.log('Campo particiones no es array:', typeof disco.particiones);
                continue;
            }

            for (const particion of disco.particiones) {
                console.log('Partición encontrada:', particion);

                if (particion.montada) {
                    console.log('Partición montada:', particion.id);

                    if (!particion.id || particion.id === '') {
                        console.log('ID vacío, saltando');
                        continue;
                    }

                    particiones.push({
                        nombre: particion.nombre || '',
                        id: particion.id,
                        disco: disco.path || ''
                    });

                    const option = document.createElement('option');
                    option.value = particion.id;
                    option.textContent = `${particion.nombre || 'Sin nombre'} (${particion.id}) - ${disco.path || 'Ruta desconocida'}`;
                    select.appendChild(option);
                    partidasEncontradas++;
                }
            }
        }

        console.log('Total de particiones encontradas:', partidasEncontradas);

        // Mostrar/ocultar el help box
        const helpBox = document.getElementById('helpBox');
        if (particiones.length === 0) {
            select.innerHTML = '<option value="">No hay particiones montadas</option>';
            console.warn('No se encontraron particiones montadas');
            if (helpBox && !localStorage.getItem('helpBoxClosed')) {
                helpBox.style.display = 'block';
            }
        } else {
            if (helpBox) helpBox.style.display = 'none';
            console.log('✅ Particiones cargadas:', particiones.length);
        }
    } catch (error) {
        console.error('❌ Error cargando particiones:', error);
        console.error('Stack:', error.stack);
        select.innerHTML = '<option value="">Error cargando</option>';
        const consola = document.getElementById('consola');
        if (consola) {
            consola.innerHTML = `<span style="color: #ff5555;">❌ Error al cargar particiones: ${error.message}</span>`;
        }
    }
}

// Actualizar ruta de guardado cuando cambia el reporte
function actualizarRutaGuardar() {
    const nombreReporte = {
        disk: "disk.svg",
        mbr: "mbr.svg",
        inode: "inode.svg",
        block: "block.svg",
        bm_inode: "bm_inode.txt",
        bm_block: "bm_block.txt",
        tree: "tree.svg",
        sb: "sb.svg"
    };

    const rutaBase = '/tmp/reportes/';
    document.getElementById('rutaGuardar').value = rutaBase + nombreReporte[reporteSeleccionado];
}

// Seleccionar tipo de reporte
document.querySelectorAll('.report-type').forEach(el => {
    el.addEventListener('click', function () {
        document.querySelectorAll('.report-type').forEach(e => e.classList.remove('active'));
        this.classList.add('active');
        reporteSeleccionado = this.dataset.reporte;
        actualizarTituloReporte();
        actualizarRutaGuardar();
    });
});

function actualizarTituloReporte() {
    const titulos = {
        'disk': '💾 Vista Previa - Disco (MBR)',
        'mbr': '📋 Vista Previa - MBR Detallado',
        'inode': '📑 Vista Previa - Inodos',
        'block': '🗂️ Vista Previa - Bloques',
        'bm_inode': '📊 Vista Previa - Bitmap Inodos',
        'bm_block': '📊 Vista Previa - Bitmap Bloques',
        'tree': '🌳 Vista Previa - Árbol de Directorios',
        'sb': '⚙️ Vista Previa - SuperBloque'
    };
    document.getElementById('reportTitleViewer').textContent = titulos[reporteSeleccionado] || '📊 Vista Previa';
}

// Generar reporte
async function generarReporte() {
    const particionId = document.getElementById('particionSelect').value;
    const rutaGuardar = document.getElementById('rutaGuardar').value;
    const consola = document.getElementById('consola');

    if (!particionId) {
        consola.innerHTML = '<span style="color: #ff5555;">❌ Error: Selecciona una partición primero</span>';
        return;
    }

    if (!rutaGuardar) {
        consola.innerHTML = '<span style="color: #ff5555;">❌ Error: Ingresa una ruta de guardado</span>';
        return;
    }

    // Mostrar loading
    document.getElementById('generarBtn').disabled = true;
    consola.innerHTML = '<div class="loading"><div class="spinner"></div>Generando reporte...</div>';

    try {
        const response = await fetch('http://localhost:3000/generar-reporte', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                tipo_reporte: reporteSeleccionado,
                particion_id: particionId,
                ruta_guardar: rutaGuardar
            })
        });

        const data = await response.json();

        // Mostrar salida en consola
        if (data.output) {
            consola.innerHTML = `<span style="color: var(--accent);">${data.output.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</span>`;
        }

        // Intentar cargar el reporte
        if (data.ruta) {
            setTimeout(() => cargarVistaPrevia(data.ruta), 500);
        }

    } catch (error) {
        consola.innerHTML = `<span style="color: #ff5555;">❌ Error: ${error.message}</span>`;
    } finally {
        document.getElementById('generarBtn').disabled = false;
    }
}

// Cargar vista previa del reporte
async function cargarVistaPrevia(ruta) {
    const viewer = document.getElementById('reportViewer');

    // Limpiar la extensión de la ruta si viene con una
    const rutaBase = ruta.includes('.') ? ruta.substring(0, ruta.lastIndexOf('.')) : ruta;

    // Determinar posibles extensiones según el tipo de reporte
    let extensiones = [];
    switch (reporteSeleccionado) {
        case 'disk':
        case 'mbr':
        case 'inode':
        case 'block':
        case 'tree':
        case 'sb':
            extensiones = ['.svg'];
            break;

        case 'bm_inode':
        case 'bm_block':
            extensiones = ['.txt'];
            break;
    }

    // Intentar cargar con cada extensión
    for (const ext of extensiones) {
        try {
            const rutaCompleta = rutaBase + ext;
            const nombreArchivo = rutaCompleta.split('/').pop();

            const response = await fetch(`http://localhost:3000/reporte/${nombreArchivo}`);

            if (response.ok) {
                const contentType = response.headers.get('content-type');
                const contenido = await response.text();

                if (contentType.includes('image')) {
                    viewer.innerHTML = `<img src="http://localhost:3000/reporte/${nombreArchivo}?${Date.now()}">`;
                    return;
                } else if (contentType.includes('svg')) {
                    viewer.innerHTML = contenido;
                    return;
                } else if (contentType.includes('text')) {
                    viewer.innerHTML = `<pre>${contenido.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</pre>`;
                    return;
                }
            }
        } catch (e) {
            // Continuar con la siguiente extensión
        }
    }

    viewer.innerHTML = '<div class="no-report">⚠️ No se pudo cargar la vista previa del reporte<br><small>Verifica que el archivo se generó correctamente</small></div>';
}

// Inicializar
window.addEventListener('load', cargarParticiones);