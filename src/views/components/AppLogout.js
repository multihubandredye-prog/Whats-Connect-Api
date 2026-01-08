export default {
    name: 'AppLogout',
    methods: {
        async handleSubmit() {
            try {
                await this.submitApi()
                showSuccessInfo("Desconectado")

                // fetch devices
                this.$emit('reload-devices')
            } catch (err) {
                showErrorInfo(err)
            }
        },

        async submitApi() {
            try {
                await http.get(`app/logout`)
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
            <div class="header">Sair</div>
            <div class="description">
                Remova sua sessão de login no aplicativo
            </div>
        </div>
    </div>
    `
}