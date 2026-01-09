export default {
    name: 'AccountChangeAvatar',
    data() {
        return {
            loading: false,
            selected_file: null,
            preview_url: null
        }
    },
    methods: {
        openModal() {
            $('#modalChangeAvatar').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            return this.selected_file !== null;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }

            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalChangeAvatar').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = new FormData();
                payload.append('avatar', $("#file_avatar")[0].files[0])

                let response = await window.http.post(`/user/avatar`, payload)
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.preview_url = null;
            this.selected_file = null;
            $("#file_avatar").val('');
        },
        handleImageChange(event) {
            const file = event.target.files[0];
            if (file) {
                this.preview_url = URL.createObjectURL(file);
                this.selected_file = file.name;
            }
        }
    },
    template: `
    <div class="olive card" @click="openModal()" style="cursor:pointer;">
        <div class="content">
            <a class="ui olive right ribbon label">Conta</a>
            <div class="header">Mudar Avatar</div>
            <div class="description">
                Atualize sua foto de perfil
            </div>
        </div>
    </div>
    
    <!--  Modal Change Avatar  -->
    <div class="ui small modal" id="modalChangeAvatar">
        <i class="close icon"></i>
        <div class="header">
            Mudar Avatar
        </div>
        <div class="content" style="max-height: 70vh; overflow-y: auto;">
            <div class="ui warning message">
                <i class="info circle icon"></i>
                Por favor, carregue uma imagem quadrada (proporção de 1:1) para evitar cortes.
                Para melhores resultados, use uma imagem de pelo menos 400x400 pixels.
            </div>
            
            <form class="ui form">
                <div class="field" style="padding-bottom: 30px">
                    <label>Imagem do Avatar</label>
                    <input type="file" style="display: none" id="file_avatar" accept="image/png,image/jpg,image/jpeg" @change="handleImageChange"/>
                    <label for="file_avatar" class="ui positive medium green left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Carregar imagem
                    </label>
                    <div v-if="preview_url" style="margin-top: 60px">
                        <img :src="preview_url" alt="Pré-visualização do Avatar" style="max-width: 100%; max-height: 300px; object-fit: contain" />
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'loading': this.loading, 'disabled': !isValidForm() || loading}"
                 @click.prevent="handleSubmit">
                Atualizar Avatar
                <i class="save icon"></i>
            </button>
        </div>
    </div>
    `
}
