#!/bin/bash

# Test de carga para 50K RPS - meli-proxy
# Usa Apache Bench (ab) que viene preinstalado en macOS

set -e

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}$1${NC}"
}

# Verificar que el sistema esté corriendo
check_system() {
    print_status "Verificando que el sistema esté activo..."
    
    if ! curl -s http://localhost:8080/sites/MLA > /dev/null; then
        print_error "El sistema no está corriendo en localhost:8080"
        echo "Por favor ejecuta: ./start.sh"
        exit 1
    fi
    
    print_status "✅ Sistema activo y respondiendo"
}

# Test básico de conectividad
connectivity_test() {
    print_header "🔗 TEST DE CONECTIVIDAD"
    
    ENDPOINTS=(
        "http://localhost:8080/sites/MLA"
        "http://localhost:8080/sites/MLA/categories"
        "http://localhost:8080/categories/MLA1051"
    )
    
    for endpoint in "${ENDPOINTS[@]}"; do
        print_status "Probando: $endpoint"
        response=$(curl -s -w "%{http_code}" -o /dev/null "$endpoint")
        if [ "$response" = "200" ] || [ "$response" = "401" ]; then
            echo -e "  ✅ Respuesta: $response"
        else
            echo -e "  ❌ Respuesta: $response"
        fi
    done
}

# Test de carga progresivo
load_test() {
    print_header "🚀 TEST DE CARGA PROGRESIVO"
    
    # Configuraciones de test
    CONFIGS=(
        "100:10:5"      # 100 requests, 10 concurrency, 5 seconds
        "1000:50:10"    # 1K requests, 50 concurrency, 10 seconds  
        "5000:100:15"   # 5K requests, 100 concurrency, 15 seconds
        "10000:200:20"  # 10K requests, 200 concurrency, 20 seconds
        "25000:500:30"  # 25K requests, 500 concurrency, 30 seconds
    )
    
    URL="http://localhost:8080/sites/MLA"
    
    for config in "${CONFIGS[@]}"; do
        IFS=':' read -r requests concurrency time <<< "$config"
        
        print_status "Test: $requests requests, $concurrency concurrencia, ${time}s duración"
        
        # Usar ab (Apache Bench) que viene con macOS
        if command -v ab &> /dev/null; then
            echo "Ejecutando: ab -n $requests -c $concurrency -t $time -r $URL"
            ab -n "$requests" -c "$concurrency" -t "$time" -r "$URL" | grep -E "(Requests per second|Time taken|Failed requests|Transfer rate)"
        else
            print_warning "Apache Bench (ab) no disponible, usando curl alternativo"
            parallel_curl_test "$requests" "$concurrency"
        fi
        
        echo ""
        sleep 2
    done
}

# Test alternativo con curl para sistemas sin ab
parallel_curl_test() {
    local total_requests=$1
    local concurrency=$2
    
    print_status "Ejecutando $total_requests requests con $concurrency procesos concurrentes..."
    
    start_time=$(date +%s.%N)
    
    # Función para ejecutar requests en paralelo
    run_batch() {
        local batch_size=$1
        for ((i=1; i<=batch_size; i++)); do
            curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/sites/MLA" &
        done
        wait
    }
    
    # Calcular batches
    requests_per_batch=$((total_requests / concurrency))
    
    for ((batch=1; batch<=concurrency; batch++)); do
        run_batch "$requests_per_batch" &
    done
    wait
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    rps=$(echo "scale=2; $total_requests / $duration" | bc)
    
    echo "Tiempo total: ${duration}s"
    echo "RPS promedio: $rps"
}

# Test de alta carga extremo
extreme_test() {
    print_header "🔥 TEST EXTREMO - TARGET 50K RPS"
    
    print_warning "Este test es intensivo. Asegúrate de monitorear:"
    echo "  - Grafana: http://localhost:3000"
    echo "  - Métricas: http://localhost:8081/metrics"
    echo ""
    
    read -p "¿Continuar con el test extremo? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Test cancelado"
        return
    fi
    
    # Test con ab si está disponible
    if command -v ab &> /dev/null; then
        print_status "🚀 Ejecutando test extremo con Apache Bench..."
        print_status "Configuración: 50,000 requests, 1000 concurrencia, 30s timeout"
        
        ab -n 50000 -c 1000 -t 30 -r "http://localhost:8080/sites/MLA"
    else
        print_status "🚀 Ejecutando test extremo con curl paralelo..."
        parallel_extreme_test
    fi
}

# Test extremo alternativo
parallel_extreme_test() {
    print_status "Configuración extrema: 30 segundos de carga máxima"
    
    # Función para generar carga continua
    generate_load() {
        local duration=30
        local end_time=$(($(date +%s) + duration))
        
        while [ $(date +%s) -lt $end_time ]; do
            for ((i=1; i<=50; i++)); do
                curl -s -o /dev/null "http://localhost:8080/sites/MLA" &
            done
            sleep 0.1
        done
        wait
    }
    
    # Ejecutar múltiples generadores en paralelo
    for ((i=1; i<=20; i++)); do
        generate_load &
    done
    
    print_status "Test en progreso... (30 segundos)"
    wait
    print_status "✅ Test extremo completado"
}

# Función principal
main() {
    print_header "===================================================="
    print_header "🚀 MELI-PROXY - TEST DE CARGA 50K RPS"
    print_header "===================================================="
    
    check_system
    echo ""
    
    connectivity_test
    echo ""
    
    load_test
    echo ""
    
    extreme_test
    echo ""
    
    print_header "📊 MONITOREO DISPONIBLE:"
    echo "  🎯 Grafana Dashboard: http://localhost:3000"
    echo "  📈 Métricas Raw: http://localhost:8081/metrics"
    echo "  📋 Logs Sistema: docker compose logs -f"
    echo ""
    print_status "✅ Test completado. Revisa Grafana para métricas detalladas."
}

# Ejecutar si se llama directamente
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
