export default {
    name: 'AppReconnect',
    methods: {
        async handleSubmit() {
            try {
                await this.submitApi()
                showSuccessInfo("Reconexão bem-sucedida")

                // buscar dispositivos
                this.$emit('reload-devices')
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            try {
                await window.http.get(`/app/reconnect`)
            } catch (error) {
                if (error.response) {
                    throw Error(error.response.data.message)
                }
                throw Error(error.message)
            }
        }
    },
    template: `
    <div class="green card" @click="handleSubmit" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">App</a>
            <div class="header">Reconectar</div>
            <div class="description">
                Por favor, reconecte-se ao serviço do WhatsApp se sua API não funcionar ou se seu aplicativo estiver inativo.
            </div>
        </div>
    </div>
    `
}